package agent

import (
	"context"
	"errors"
	"time"

	"github.com/parag-labs/incident-commander/internal/agent/llm"
	"github.com/parag-labs/incident-commander/internal/incident"
	"github.com/parag-labs/incident-commander/internal/policy"
	"github.com/parag-labs/incident-commander/internal/simulator"
	"github.com/parag-labs/incident-commander/internal/tools"
	"github.com/parag-labs/incident-commander/pkg/models"
)

// Options configures one investigation run.
type Options struct {
	Workers           int  // evidence-collection pool size
	AutoApproveMedium bool // let MEDIUM remediations run without a human
	MaxToolCalls      int  // hard bound on total tool calls (safety rail)
	Now               func() time.Time
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now().UTC()
}

// Runner ties the deterministic core and the model together for a single incident. It
// is safe to reuse across incidents; it holds no per-incident state.
type Runner struct {
	registry  *tools.Registry
	collector *Collector
	model     llm.LLMClient
}

// NewRunner builds a runner from a tool registry and a model client.
func NewRunner(registry *tools.Registry, model llm.LLMClient, workers int) *Runner {
	return &Runner{
		registry:  registry,
		collector: NewCollector(registry, workers),
		model:     model,
	}
}

// Result is the outcome of a run, including the finished incident and its report.
type Result struct {
	Incident *models.Incident
	Report   models.IncidentReport
	// Rejected is set when the model's output failed validation (nil on success).
	Rejected error
}

// Run drives one incident from an alert to a report: triage, concurrent investigation,
// validated diagnosis, deterministic risk gating, optional remediation, and
// verification. The state machine is advanced only through legal transitions; the model
// never sets state, and no action runs that the policy engine hasn't cleared.
func (r *Runner) Run(ctx context.Context, env *simulator.Env, inc *models.Incident, opts Options) (Result, error) {
	now := opts.now

	if err := incident.Transition(inc, models.StateTriaging, "beginning triage", now()); err != nil {
		return Result{Incident: inc}, err
	}

	// --- INVESTIGATION (concurrent) ---
	if err := incident.Transition(inc, models.StateInvestigating, "collecting evidence", now()); err != nil {
		return Result{Incident: inc}, err
	}
	evidence, calls := r.collector.Collect(ctx, env, inc.Alert.Service, now)
	if opts.MaxToolCalls > 0 && len(calls) > opts.MaxToolCalls {
		calls = calls[:opts.MaxToolCalls]
	}
	inc.Evidence = evidence
	inc.ToolCalls = calls
	if len(evidence) == 0 {
		_ = incident.Transition(inc, models.StateInvestigationFailed, "no evidence collected", now())
		return Result{Incident: inc, Report: BuildReport(inc)}, nil
	}

	// --- DIAGNOSIS (model proposes, we validate) ---
	bundle := llm.EvidenceBundle{
		IncidentID:  inc.ID,
		Alert:       inc.Alert,
		Evidence:    evidence,
		Deployments: deployNotes(env),
	}
	raw, err := r.model.Diagnose(ctx, bundle)
	if err != nil {
		_ = incident.Transition(inc, models.StateInvestigationFailed, "model error: "+err.Error(), now())
		return Result{Incident: inc, Report: BuildReport(inc)}, nil
	}
	diag, err := Validate(raw, evidence, r.registry)
	if err != nil {
		// A rejected model response is a safe outcome, not a crash: we escalate to a
		// human rather than act on output we couldn't trust.
		_ = incident.Transition(inc, models.StateInvestigating, "model output rejected", now())
		_ = incident.Transition(inc, models.StateInvestigationFailed, "rejected: "+err.Error(), now())
		res := Result{Incident: inc, Rejected: err}
		res.Report = BuildReport(inc)
		return res, nil
	}
	inc.Diagnosis = &diag
	if err := incident.Transition(inc, models.StateRootCauseIdentified, "diagnosis validated", now()); err != nil {
		return Result{Incident: inc}, err
	}

	// No recommended action: this is an informational incident. Resolve read-only.
	if diag.RecommendedAction == "" {
		_ = incident.Transition(inc, models.StateResolved, "informational; no remediation required", now())
		return Result{Incident: inc, Report: BuildReport(inc)}, nil
	}

	// --- RISK GATING (deterministic) ---
	safety, _ := r.registry.SafetyOf(diag.RecommendedAction)
	rem := models.Remediation{
		Action:  diag.RecommendedAction,
		Service: diag.RecommendedService,
		Safety:  safety,
		Reason:  diag.Summary,
	}
	ra := policy.Assess(rem, safety, policy.Config{AutoApproveMedium: opts.AutoApproveMedium})
	inc.Remediation = &rem
	inc.Risk = &ra
	if err := incident.Transition(inc, models.StateRemediationProposed, "remediation proposed", now()); err != nil {
		return Result{Incident: inc}, err
	}

	// If a human is required, we stop at WAITING_FOR_APPROVAL - the agent never escalates
	// itself past the gate.
	if !ra.AutoApprovable {
		_ = incident.Transition(inc, models.StateWaitingForApproval, ra.Reason, now())
		return Result{Incident: inc, Report: BuildReport(inc)}, nil
	}

	// --- REMEDIATION + VERIFICATION ---
	return r.remediateAndVerify(ctx, env, inc, rem, opts)
}

// remediateAndVerify executes an auto-approved remediation and checks it helped.
func (r *Runner) remediateAndVerify(ctx context.Context, env *simulator.Env, inc *models.Incident, rem models.Remediation, opts Options) (Result, error) {
	now := opts.now
	before := verifyMetrics(env, inc.Alert.Service)

	if err := incident.Transition(inc, models.StateRemediating, "executing remediation", now()); err != nil {
		return Result{Incident: inc}, err
	}
	caps := tools.Capabilities{AllowAutoMedium: opts.AutoApproveMedium}
	improved, call, err := r.registry.Remediate(ctx, env, rem.Action, rem.Service, caps, now)
	inc.ToolCalls = append(inc.ToolCalls, call)
	if err != nil {
		_ = incident.Transition(inc, models.StateRemediationFailed, "remediation error: "+err.Error(), now())
		return Result{Incident: inc, Report: BuildReport(inc)}, nil
	}

	if err := incident.Transition(inc, models.StateVerifying, "verifying remediation", now()); err != nil {
		return Result{Incident: inc}, err
	}
	after := verifyMetrics(env, inc.Alert.Service)
	vr := models.VerificationResult{
		Improved:     improved && after.errPct <= before.errPct && after.p99 <= before.p99,
		BeforeErrPct: before.errPct, AfterErrPct: after.errPct,
		BeforeP99MS: before.p99, AfterP99MS: after.p99,
	}
	if vr.Improved {
		vr.Note = "metrics recovered to baseline after remediation"
	} else {
		vr.Note = "metrics did not improve; remediation ineffective"
	}
	inc.Verification = &vr

	if vr.Improved {
		_ = incident.Transition(inc, models.StateResolved, "verified improved", now())
	} else {
		_ = incident.Transition(inc, models.StateVerificationFailed, "no improvement after remediation", now())
	}
	return Result{Incident: inc, Report: BuildReport(inc)}, nil
}

type metricPair struct{ errPct, p99 float64 }

func verifyMetrics(env *simulator.Env, svc models.Service) metricPair {
	m := env.Snapshot(svc)
	return metricPair{errPct: m.ErrorRatePct, p99: m.LatencyP99MS}
}

func deployNotes(env *simulator.Env) []llm.DeployNote {
	var out []llm.DeployNote
	for _, d := range env.Deployments() {
		out = append(out, llm.DeployNote{Service: d.Service, Version: d.Version, MinutesAgo: d.MinsAgo})
	}
	return out
}

// BuildReport assembles the evidence-backed incident report from the incident's
// current state. It's a pure projection - safe to call at any point in the lifecycle.
func BuildReport(inc *models.Incident) models.IncidentReport {
	rep := models.IncidentReport{
		IncidentID:   inc.ID,
		Title:        inc.Title,
		State:        inc.State,
		Evidence:     inc.Evidence,
		ToolCalls:    inc.ToolCalls,
		Remediation:  inc.Remediation,
		Verification: inc.Verification,
		GeneratedAt:  inc.UpdatedAt,
	}
	if inc.Diagnosis != nil {
		rep.Diagnosis = *inc.Diagnosis
		if top, ok := inc.Diagnosis.Top(); ok {
			rep.RootCause = top.Cause
		}
	}
	return rep
}

// Approve executes a remediation that was waiting for human approval. It is the only
// path that runs a HIGH-risk action, and only because a human explicitly asked for it.
// The incident must be in WAITING_FOR_APPROVAL with a proposed remediation.
func (r *Runner) Approve(ctx context.Context, env *simulator.Env, inc *models.Incident, opts Options) (Result, error) {
	if inc.State != models.StateWaitingForApproval || inc.Remediation == nil {
		return Result{Incident: inc}, ErrNotAwaitingApproval
	}
	rem := *inc.Remediation
	now := opts.now
	before := verifyMetrics(env, inc.Alert.Service)

	if err := incident.Transition(inc, models.StateRemediating, "human approved; executing", now()); err != nil {
		return Result{Incident: inc}, err
	}
	improved, call, err := r.registry.Remediate(ctx, env, rem.Action, rem.Service,
		tools.Capabilities{HumanApproved: true}, now)
	inc.ToolCalls = append(inc.ToolCalls, call)
	if err != nil {
		_ = incident.Transition(inc, models.StateRemediationFailed, "remediation error: "+err.Error(), now())
		return Result{Incident: inc, Report: BuildReport(inc)}, nil
	}

	if err := incident.Transition(inc, models.StateVerifying, "verifying remediation", now()); err != nil {
		return Result{Incident: inc}, err
	}
	after := verifyMetrics(env, inc.Alert.Service)
	vr := models.VerificationResult{
		Improved:     improved && after.errPct <= before.errPct && after.p99 <= before.p99,
		BeforeErrPct: before.errPct, AfterErrPct: after.errPct,
		BeforeP99MS: before.p99, AfterP99MS: after.p99,
	}
	if vr.Improved {
		vr.Note = "metrics recovered to baseline after remediation"
	} else {
		vr.Note = "metrics did not improve; remediation ineffective"
	}
	inc.Verification = &vr
	if vr.Improved {
		_ = incident.Transition(inc, models.StateResolved, "verified improved", now())
	} else {
		_ = incident.Transition(inc, models.StateVerificationFailed, "no improvement after remediation", now())
	}
	return Result{Incident: inc, Report: BuildReport(inc)}, nil
}

// ErrNoModel is returned when a runner is built without a model client.
var ErrNoModel = errors.New("no model client configured")

// ErrNotAwaitingApproval is returned when Approve is called on an incident that isn't
// waiting for a human decision.
var ErrNotAwaitingApproval = errors.New("incident is not awaiting approval")
