package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/parag-labs/incident-commander/pkg/models"
)

func ev(id string, svc models.Service, detail map[string]any) models.Evidence {
	return models.Evidence{ID: id, Service: svc, Detail: detail}
}

func diagnose(t *testing.T, bundle EvidenceBundle) models.Diagnosis {
	t.Helper()
	raw, err := MockLLM{}.Diagnose(context.Background(), bundle)
	if err != nil {
		t.Fatalf("diagnose error: %v", err)
	}
	var d models.Diagnosis
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("mock produced invalid JSON: %v", err)
	}
	return d
}

func TestMockPicksDatabaseConnectionExhaustion(t *testing.T) {
	d := diagnose(t, EvidenceBundle{
		Evidence: []models.Evidence{
			ev("e1", models.ServiceDatabase, map[string]any{"db_connections": 100, "latency_p99_ms": 2500.0, "error_rate_pct": 18.0, "healthy": false}),
			ev("e2", models.ServiceQueue, map[string]any{"queue_depth": 20, "healthy": true}),
		},
	})
	if d.RecommendedAction != "reduce_db_connections" || d.RecommendedService != models.ServiceDatabase {
		t.Fatalf("want reduce_db_connections on database, got %q on %s", d.RecommendedAction, d.RecommendedService)
	}
	top, ok := d.Top()
	if !ok || len(top.EvidenceIDs) == 0 {
		t.Fatal("expected a hypothesis citing evidence")
	}
}

func TestMockPicksCacheFlushOnCollapse(t *testing.T) {
	d := diagnose(t, EvidenceBundle{
		Evidence: []models.Evidence{
			ev("e1", models.ServiceCache, map[string]any{"cache_hit_rate_pct": 8.0, "latency_p99_ms": 400.0, "healthy": false}),
		},
	})
	if d.RecommendedAction != "clear_cache" {
		t.Fatalf("want clear_cache, got %q", d.RecommendedAction)
	}
}

func TestMockRollsBackWhenErrorFollowsDeploy(t *testing.T) {
	d := diagnose(t, EvidenceBundle{
		Alert: models.Alert{Service: models.ServiceAPI},
		Evidence: []models.Evidence{
			ev("e1", models.ServiceAPI, map[string]any{"error_rate_pct": 22.0, "latency_p99_ms": 600.0, "healthy": false}),
		},
		Deployments: []DeployNote{{Service: models.ServiceAPI, Version: "v2.4.1", MinutesAgo: 6}},
	})
	if d.RecommendedAction != "rollback_deployment" {
		t.Fatalf("want rollback_deployment, got %q", d.RecommendedAction)
	}
	if d.Risk != models.RiskHigh || !d.NeedsHumanApproval {
		t.Fatalf("a rollback must be high-risk and need approval, got risk=%s approval=%v", d.Risk, d.NeedsHumanApproval)
	}
}

func TestMockOnlyCitesEvidenceItWasGiven(t *testing.T) {
	bundle := EvidenceBundle{
		Evidence: []models.Evidence{
			ev("e1", models.ServiceDatabase, map[string]any{"db_connections": 100, "healthy": false}),
			ev("e2", models.ServiceDatabase, map[string]any{"latency_p99_ms": 2500.0, "healthy": false}),
		},
	}
	d := diagnose(t, bundle)
	given := map[string]bool{"e1": true, "e2": true}
	top, _ := d.Top()
	for _, id := range top.EvidenceIDs {
		if !given[id] {
			t.Fatalf("mock cited an evidence ID it was never given: %q", id)
		}
	}
}

func TestMockReportsLowConfidenceWhenEverythingHealthy(t *testing.T) {
	d := diagnose(t, EvidenceBundle{
		Evidence: []models.Evidence{
			ev("e1", models.ServiceAPI, map[string]any{"error_rate_pct": 0.2, "healthy": true}),
			ev("e2", models.ServiceQueue, map[string]any{"queue_depth": 20, "healthy": true}),
		},
	})
	if d.RecommendedAction != "" || !d.NeedsHumanApproval {
		t.Fatalf("with no anomaly the mock should recommend nothing and defer to a human, got %+v", d)
	}
}

func TestMockExplainMentionsRootCause(t *testing.T) {
	report := models.IncidentReport{IncidentID: "inc-1", Title: "t", RootCause: "database pool exhausted"}
	got, err := MockLLM{}.Explain(context.Background(), report)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" || !contains(got, "database pool exhausted") {
		t.Fatalf("explanation should mention the root cause, got %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
