// Package tools implements the structured capabilities the agent may use. Read-only
// investigation tools return grounded Evidence; remediation tools change the simulated
// environment. Every tool declares a safety level, runs under a context deadline, is
// authorization-checked before execution, and produces an audit record. The AI never
// touches the environment directly - it can only ask for a named tool by producing a
// recommendation the deterministic layer then validates and runs.
package tools

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/parag-labs/incident-commander/internal/simulator"
	"github.com/parag-labs/incident-commander/pkg/models"
)

// Investigator is a read-only tool that collects evidence about a service.
type Investigator struct {
	Name   string
	Safety models.SafetyLevel // always low
	Desc   string
	fn     func(env *simulator.Env, svc models.Service, id func() string) []models.Evidence
}

// Remediator is a state-changing tool. Its safety level gates whether it may run
// automatically.
type Remediator struct {
	Name   string
	Safety models.SafetyLevel
	Desc   string
	fn     func(env *simulator.Env, svc models.Service) bool
}

// Capabilities is the authorization envelope a run executes under.
type Capabilities struct {
	// AllowAutoMedium permits MEDIUM-safety remediations to run without a human.
	AllowAutoMedium bool
	// HumanApproved authorizes any safety level - it is set only after an explicit
	// human approval, and is the single path by which a HIGH action can ever run.
	HumanApproved bool
}

// ErrUnauthorized is returned when a remediation's safety level exceeds what the
// current capabilities permit to run automatically.
type ErrUnauthorized struct {
	Tool   string
	Safety models.SafetyLevel
}

func (e ErrUnauthorized) Error() string {
	return fmt.Sprintf("tool %q (%s) is not authorized to run automatically", e.Tool, e.Safety)
}

// ErrUnknownTool is returned when a name that isn't a registered tool is requested -
// the guard against a hallucinated tool name from the model.
type ErrUnknownTool struct{ Name string }

func (e ErrUnknownTool) Error() string { return fmt.Sprintf("unknown tool %q", e.Name) }

// Registry holds every tool the system exposes.
type Registry struct {
	investigators map[string]Investigator
	remediators   map[string]Remediator
}

// NewRegistry builds the default tool set.
func NewRegistry() *Registry {
	r := &Registry{
		investigators: map[string]Investigator{},
		remediators:   map[string]Remediator{},
	}
	for _, inv := range defaultInvestigators() {
		r.investigators[inv.Name] = inv
	}
	for _, rem := range defaultRemediators() {
		r.remediators[rem.Name] = rem
	}
	return r
}

// InvestigatorNames returns the read-only tool names in a stable order.
func (r *Registry) InvestigatorNames() []string {
	names := make([]string, 0, len(r.investigators))
	for n := range r.investigators {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// IsRemediator reports whether name is a known remediation tool.
func (r *Registry) IsRemediator(name string) bool {
	_, ok := r.remediators[name]
	return ok
}

// SafetyOf returns the declared safety level of a remediation tool.
func (r *Registry) SafetyOf(name string) (models.SafetyLevel, bool) {
	rem, ok := r.remediators[name]
	if !ok {
		return "", false
	}
	return rem.Safety, true
}

// Collect runs one investigation tool under a deadline and returns its evidence plus an
// audit record. A cancelled context aborts before the tool runs.
func (r *Registry) Collect(ctx context.Context, env *simulator.Env, name string, svc models.Service, id func() string, now func() time.Time) ([]models.Evidence, models.ToolCall, error) {
	call := models.ToolCall{Tool: name, Service: svc, Safety: models.SafetyLow, StartedAt: now()}
	inv, ok := r.investigators[name]
	if !ok {
		call.Error = ErrUnknownTool{name}.Error()
		return nil, call, ErrUnknownTool{name}
	}
	if err := ctx.Err(); err != nil {
		call.Error = err.Error()
		return nil, call, err
	}
	ev := inv.fn(env, svc, id)
	call.OK = true
	call.DurationMS = time.Since(call.StartedAt).Milliseconds()
	return ev, call, nil
}

// Remediate runs a remediation tool after checking authorization. It returns whether
// the action improved the environment, plus an audit record. HIGH-safety tools are
// never authorized to run here; MEDIUM only when capabilities allow it.
func (r *Registry) Remediate(ctx context.Context, env *simulator.Env, name string, svc models.Service, caps Capabilities, now func() time.Time) (bool, models.ToolCall, error) {
	rem, ok := r.remediators[name]
	call := models.ToolCall{Tool: name, Service: svc, StartedAt: now()}
	if !ok {
		call.Error = ErrUnknownTool{name}.Error()
		return false, call, ErrUnknownTool{name}
	}
	call.Safety = rem.Safety
	if !authorized(rem.Safety, caps) {
		call.Error = ErrUnauthorized{name, rem.Safety}.Error()
		return false, call, ErrUnauthorized{name, rem.Safety}
	}
	if err := ctx.Err(); err != nil {
		call.Error = err.Error()
		return false, call, err
	}
	improved := rem.fn(env, svc)
	call.OK = true
	call.DurationMS = time.Since(call.StartedAt).Milliseconds()
	return improved, call, nil
}

// authorized reports whether a safety level may run under caps. An explicit human
// approval authorizes any level; otherwise LOW always runs, MEDIUM runs only when
// auto-approval is on, and HIGH never runs automatically.
func authorized(s models.SafetyLevel, caps Capabilities) bool {
	if caps.HumanApproved {
		return true
	}
	switch s {
	case models.SafetyLow:
		return true
	case models.SafetyMedium:
		return caps.AllowAutoMedium
	default: // high
		return false
	}
}
