package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/parag-labs/incident-commander/pkg/models"
)

// MockLLM is the deterministic stand-in for a real model, used in every test and in CI.
// It is a rule-based root-cause reasoner: it groups the evidence by service, scores each
// unhealthy service by how far its metrics deviate from a healthy baseline, picks the
// worst as the root cause, and maps the dominant symptom to a remediation. Crucially it
// only ever cites evidence IDs it was actually given, so it can't hallucinate - which is
// exactly the property the agent's validation layer also enforces on a real model.
type MockLLM struct{}

// Name identifies the provider.
func (MockLLM) Name() string { return "mock" }

// baseline metrics a healthy service sits at.
type baseline struct{ latency, errRate, cpu, mem, queue, dbConns, cacheHit float64 }

func baselineFor(svc models.Service) baseline {
	b := baseline{latency: 120, errRate: 0.2, cpu: 30, mem: 40, queue: 20, dbConns: 35, cacheHit: 94}
	if svc == models.ServiceCache {
		b.latency = 5
	}
	return b
}

// serviceView is the merged metric picture for one service, plus its backing evidence.
type serviceView struct {
	svc         models.Service
	m           map[string]float64
	healthy     bool
	haveHealthy bool
	evidenceIDs []string
}

// Diagnose reasons over the bundle and returns a JSON models.Diagnosis.
func (MockLLM) Diagnose(_ context.Context, bundle EvidenceBundle) (string, error) {
	views := groupByService(bundle.Evidence)

	worst, score := pickWorst(views)
	if worst == nil {
		// Nothing looks unhealthy: report low confidence, no action, defer to a human.
		d := models.Diagnosis{
			Summary:            "no service shows a clear anomaly in the collected evidence",
			Hypotheses:         nil,
			RecommendedAction:  "",
			Risk:               models.RiskLow,
			NeedsHumanApproval: true,
		}
		return marshal(d)
	}

	action := chooseAction(worst, bundle.Deployments)
	risk, needsApproval := riskFor(action)
	confidence := confidenceFromScore(score)

	d := models.Diagnosis{
		Summary: fmt.Sprintf("%s is the most degraded service; its metrics deviate furthest from baseline", worst.svc),
		Hypotheses: []models.Hypothesis{{
			Cause:       causeFor(worst),
			Confidence:  confidence,
			EvidenceIDs: worst.evidenceIDs,
		}},
		RecommendedAction:  action,
		RecommendedService: worst.svc,
		Risk:               risk,
		NeedsHumanApproval: needsApproval,
	}
	return marshal(d)
}

// Explain produces a deterministic narrative from a finished report.
func (MockLLM) Explain(_ context.Context, report models.IncidentReport) (string, error) {
	verdict := "was not remediated automatically"
	if report.Verification != nil && report.Verification.Improved {
		verdict = "was remediated and verified as improved"
	} else if report.Verification != nil {
		verdict = "was remediated but verification did not confirm improvement"
	}
	act := "no action was taken"
	if report.Remediation != nil {
		act = fmt.Sprintf("the recommended action was %q on %s", report.Remediation.Action, report.Remediation.Service)
	}
	return fmt.Sprintf(
		"Incident %s (%s). Root cause: %s. Based on %d pieces of evidence, %s; the incident %s.",
		report.IncidentID, report.Title, report.RootCause, len(report.Evidence), act, verdict,
	), nil
}

func groupByService(evidence []models.Evidence) map[models.Service]*serviceView {
	views := map[models.Service]*serviceView{}
	for _, e := range evidence {
		v := views[e.Service]
		if v == nil {
			v = &serviceView{svc: e.Service, m: map[string]float64{}}
			views[e.Service] = v
		}
		v.evidenceIDs = append(v.evidenceIDs, e.ID)
		for k, raw := range e.Detail {
			if f, ok := toFloat(raw); ok {
				v.m[k] = f
			}
			if k == "healthy" {
				if b, ok := raw.(bool); ok {
					v.healthy, v.haveHealthy = b, true
				}
			}
		}
	}
	return views
}

// pickWorst returns the unhealthy service whose metrics deviate most from baseline.
func pickWorst(views map[models.Service]*serviceView) (*serviceView, float64) {
	// Stable iteration for deterministic tie-breaking.
	svcs := make([]models.Service, 0, len(views))
	for s := range views {
		svcs = append(svcs, s)
	}
	sort.Slice(svcs, func(i, j int) bool { return orderOf(svcs[i]) < orderOf(svcs[j]) })

	var worst *serviceView
	best := 0.0
	for _, s := range svcs {
		v := views[s]
		// Only consider services that report unhealthy, when that signal is present.
		if v.haveHealthy && v.healthy {
			continue
		}
		sc := deviationScore(v)
		if sc > best {
			best, worst = sc, v
		}
	}
	return worst, best
}

func deviationScore(v *serviceView) float64 {
	b := baselineFor(v.svc)
	score := 0.0
	if lat, ok := v.m["latency_p99_ms"]; ok && b.latency > 0 {
		score += rel(lat, b.latency)
	}
	if er, ok := v.m["error_rate_pct"]; ok {
		score += er - b.errRate
	}
	if c, ok := v.m["cpu_pct"]; ok {
		score += rel(c, b.cpu)
	}
	if mem, ok := v.m["memory_pct"]; ok {
		score += rel(mem, b.mem)
	}
	if q, ok := v.m["queue_depth"]; ok {
		score += rel(q, b.queue)
	}
	if dc, ok := v.m["db_connections"]; ok {
		score += rel(dc, b.dbConns)
	}
	if ch, ok := v.m["cache_hit_rate_pct"]; ok && ch < b.cacheHit {
		score += (b.cacheHit - ch) / 5 // a collapsed hit rate is a strong signal
	}
	return score
}

// chooseAction maps the worst service's dominant symptom to a remediation tool name.
func chooseAction(v *serviceView, deploys []DeployNote) string {
	switch {
	case v.m["db_connections"] >= 90:
		return "reduce_db_connections"
	case has(v.m, "cache_hit_rate_pct") && v.m["cache_hit_rate_pct"] < 50:
		return "clear_cache"
	case v.m["error_rate_pct"] >= 5 && deployedRecently(v.svc, deploys):
		return "rollback_deployment"
	case v.m["memory_pct"] >= 90:
		return "restart_service"
	case v.m["cpu_pct"] >= 90:
		return "scale_service"
	case v.m["queue_depth"] >= 500:
		return "scale_service"
	default:
		// Elevated latency or errors with no more specific signal: a restart is the
		// least-surprising recoverable action.
		return "restart_service"
	}
}

func causeFor(v *serviceView) string {
	switch {
	case v.m["db_connections"] >= 90:
		return fmt.Sprintf("%s connection pool exhausted (connections at %.0f)", v.svc, v.m["db_connections"])
	case has(v.m, "cache_hit_rate_pct") && v.m["cache_hit_rate_pct"] < 50:
		return fmt.Sprintf("%s hit rate collapsed to %.0f%%", v.svc, v.m["cache_hit_rate_pct"])
	case v.m["memory_pct"] >= 90:
		return fmt.Sprintf("%s memory saturated at %.0f%%", v.svc, v.m["memory_pct"])
	case v.m["cpu_pct"] >= 90:
		return fmt.Sprintf("%s CPU saturated at %.0f%%", v.svc, v.m["cpu_pct"])
	case v.m["queue_depth"] >= 500:
		return fmt.Sprintf("%s queue backed up to %.0f", v.svc, v.m["queue_depth"])
	case v.m["error_rate_pct"] >= 5:
		return fmt.Sprintf("%s error rate elevated to %.1f%%", v.svc, v.m["error_rate_pct"])
	default:
		return fmt.Sprintf("%s latency elevated to %.0fms", v.svc, v.m["latency_p99_ms"])
	}
}

func riskFor(action string) (models.Risk, bool) {
	switch action {
	case "rollback_deployment":
		return models.RiskHigh, true
	case "":
		return models.RiskLow, true
	default:
		return models.RiskMedium, false
	}
}

func confidenceFromScore(score float64) float64 {
	// Squash an unbounded deviation score into a readable 0.5..0.98 confidence.
	c := 0.5 + score/100
	if c > 0.98 {
		c = 0.98
	}
	if c < 0.5 {
		c = 0.5
	}
	// Round to two decimals for stable output.
	return float64(int(c*100+0.5)) / 100
}

func deployedRecently(svc models.Service, deploys []DeployNote) bool {
	for _, d := range deploys {
		if d.Service == svc && d.MinutesAgo <= 30 {
			return true
		}
	}
	return false
}

func rel(v, base float64) float64 {
	if base == 0 {
		return v
	}
	d := (v - base) / base
	if d < 0 {
		return 0
	}
	return d
}

func has(m map[string]float64, k string) bool { _, ok := m[k]; return ok }

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func orderOf(s models.Service) int {
	for i, x := range models.AllServices {
		if x == s {
			return i
		}
	}
	return len(models.AllServices)
}

func marshal(d models.Diagnosis) (string, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
