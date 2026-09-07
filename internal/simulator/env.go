// Package simulator is the deterministic stand-in for a production environment. It is
// the reason this project can prove behaviour without touching real infrastructure:
// every scenario produces exactly the same metrics every time, and remediation actions
// change that state in exactly the same way. No randomness, no wall clock, no network.
package simulator

import (
	"sort"
	"sync"

	"github.com/parag-labs/incident-commander/pkg/models"
)

// Metrics is one service's health snapshot. Fields are the signals the investigation
// tools read and the verifier compares before/after.
type Metrics struct {
	LatencyP99MS float64 `json:"latency_p99_ms"`
	ErrorRatePct float64 `json:"error_rate_pct"`
	CPUPct       float64 `json:"cpu_pct"`
	MemoryPct    float64 `json:"memory_pct"`
	QueueDepth   int     `json:"queue_depth"`
	DBConns      int     `json:"db_connections"`
	CacheHitPct  float64 `json:"cache_hit_rate_pct"`
	Healthy      bool    `json:"healthy"`
}

// Deployment is a recent release, used by the "recent deployments" tool.
type Deployment struct {
	Service  models.Service `json:"service"`
	Version  string         `json:"version"`
	MinsAgo  int            `json:"minutes_ago"`
	Rollback bool           `json:"is_rollback"`
}

// baseline is the healthy steady state for every service.
func baseline(s models.Service) Metrics {
	m := Metrics{
		LatencyP99MS: 120,
		ErrorRatePct: 0.2,
		CPUPct:       30,
		MemoryPct:    40,
		QueueDepth:   20,
		DBConns:      35,
		CacheHitPct:  94,
		Healthy:      true,
	}
	if s == models.ServiceCache {
		m.LatencyP99MS = 5
	}
	return m
}

// dependencies is the static service dependency graph the agent can inspect.
var dependencies = map[models.Service][]models.Service{
	models.ServiceAPI:      {models.ServicePayment, models.ServiceDatabase, models.ServiceCache},
	models.ServicePayment:  {models.ServiceDatabase},
	models.ServiceDatabase: {},
	models.ServiceCache:    {},
	models.ServiceQueue:    {models.ServiceDatabase},
}

// Env is the simulated environment. It is safe for concurrent reads (the tools collect
// evidence from several goroutines at once) and serialises writes from remediation.
type Env struct {
	mu       sync.RWMutex
	scenario Scenario
	metrics  map[models.Service]Metrics
	deploys  []Deployment
	fixed    bool // whether the active fault has been remediated
}

// New builds an environment with the given scenario applied.
func New(sc Scenario) *Env {
	e := &Env{scenario: sc, metrics: map[models.Service]Metrics{}}
	for _, s := range models.AllServices {
		e.metrics[s] = baseline(s)
	}
	e.deploys = sc.Deployments
	e.applyFault()
	return e
}

// applyFault degrades the affected services according to the scenario.
func (e *Env) applyFault() {
	for svc, override := range e.scenario.Faults {
		m := e.metrics[svc]
		override(&m)
		e.metrics[svc] = m
	}
}

// Scenario returns the active scenario's metadata.
func (e *Env) Scenario() Scenario { return e.scenario }

// Snapshot returns a copy of one service's metrics.
func (e *Env) Snapshot(s models.Service) Metrics {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.metrics[s]
}

// All returns a copy of every service's metrics, keyed by service.
func (e *Env) All() map[models.Service]Metrics {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make(map[models.Service]Metrics, len(e.metrics))
	for k, v := range e.metrics {
		out[k] = v
	}
	return out
}

// Deployments returns recent deployments, most recent first.
func (e *Env) Deployments() []Deployment {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := append([]Deployment(nil), e.deploys...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].MinsAgo < out[j].MinsAgo })
	return out
}

// Dependencies returns the direct dependencies of a service.
func (e *Env) Dependencies(s models.Service) []models.Service {
	return append([]models.Service(nil), dependencies[s]...)
}

// Apply runs a remediation action against a service and reports whether it helped.
// Only the scenario's correct (action, service) pair clears the fault; anything else
// is a no-op that leaves the environment degraded - which is exactly how the verifier
// catches a wrong remediation.
func (e *Env) Apply(action string, svc models.Service) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.fixed {
		return true
	}
	if action == e.scenario.FixAction && svc == e.scenario.FixService {
		for s := range e.scenario.Faults {
			e.metrics[s] = baseline(s)
		}
		if e.scenario.Deployments != nil && action == "rollback_deployment" {
			e.deploys = append([]Deployment{{Service: svc, Version: "rolled-back", MinsAgo: 0, Rollback: true}}, e.deploys...)
		}
		e.fixed = true
		return true
	}
	return false
}

// Fixed reports whether the active fault has been remediated.
func (e *Env) Fixed() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.fixed
}
