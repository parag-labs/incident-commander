package agent

import (
	"context"
	"testing"
	"time"

	"github.com/parag-labs/incident-commander/internal/agent/llm"
	"github.com/parag-labs/incident-commander/internal/incident"
	"github.com/parag-labs/incident-commander/internal/simulator"
	"github.com/parag-labs/incident-commander/internal/tools"
	"github.com/parag-labs/incident-commander/pkg/models"
)

func fixedNow() time.Time { return time.Unix(0, 0).UTC() }

func runScenario(t *testing.T, id string, autoMedium bool) Result {
	t.Helper()
	sc, ok := simulator.ScenarioByID(id)
	if !ok {
		t.Fatalf("scenario %s not found", id)
	}
	env := simulator.New(sc)
	inc := incident.New("inc-"+id, sc.Alert, fixedNow())
	runner := NewRunner(tools.NewRegistry(), llm.MockLLM{}, 4)
	res, err := runner.Run(context.Background(), env, inc, Options{
		Workers: 4, AutoApproveMedium: autoMedium, MaxToolCalls: 40, Now: fixedNow,
	})
	if err != nil {
		t.Fatalf("run error for %s: %v", id, err)
	}
	return res
}

func TestEndToEndResolvesAutoApprovableIncidents(t *testing.T) {
	// With MEDIUM auto-approval on, every non-rollback scenario should resolve.
	autoResolvable := []string{
		"database_overload", "cache_failure", "queue_backlog", "memory_leak",
		"dependency_timeout", "high_cpu", "network_latency", "cascading_failure",
	}
	for _, id := range autoResolvable {
		res := runScenario(t, id, true)
		if res.Rejected != nil {
			t.Fatalf("%s: diagnosis unexpectedly rejected: %v", id, res.Rejected)
		}
		if res.Incident.State != models.StateResolved {
			t.Fatalf("%s: want RESOLVED, got %s", id, res.Incident.State)
		}
		if res.Incident.Verification == nil || !res.Incident.Verification.Improved {
			t.Fatalf("%s: expected verified improvement", id)
		}
	}
}

func TestRootCauseServiceMatchesGroundTruth(t *testing.T) {
	for _, sc := range simulator.Scenarios {
		res := runScenario(t, sc.ID, true)
		if res.Incident.Remediation == nil {
			// rollback scenarios stop for approval; check those separately
			continue
		}
		if res.Incident.Remediation.Action != sc.FixAction || res.Incident.Remediation.Service != sc.FixService {
			t.Errorf("%s: agent chose %s on %s, ground truth is %s on %s",
				sc.ID, res.Incident.Remediation.Action, res.Incident.Remediation.Service, sc.FixAction, sc.FixService)
		}
	}
}

func TestHighRiskRollbackStopsForApproval(t *testing.T) {
	// deployment_regression and auth_failure both require a HIGH rollback: the agent must
	// stop at WAITING_FOR_APPROVAL and never execute automatically.
	for _, id := range []string{"deployment_regression", "auth_failure"} {
		res := runScenario(t, id, true) // even with auto-approve MEDIUM on
		if res.Incident.State != models.StateWaitingForApproval {
			t.Fatalf("%s: HIGH rollback must wait for approval, got %s", id, res.Incident.State)
		}
		if res.Incident.Risk == nil || res.Incident.Risk.AutoApprovable {
			t.Fatalf("%s: rollback must not be auto-approvable", id)
		}
		// The environment must be untouched - nothing executed.
		if res.Incident.Verification != nil {
			t.Fatalf("%s: no remediation should have run", id)
		}
	}
}

func TestWithoutAutoApprovalMediumWaits(t *testing.T) {
	res := runScenario(t, "database_overload", false)
	if res.Incident.State != models.StateWaitingForApproval {
		t.Fatalf("MEDIUM without auto-approval should wait, got %s", res.Incident.State)
	}
}

func TestRejectedModelOutputFailsSafe(t *testing.T) {
	// A model that hallucinates a tool must never touch the environment.
	sc, _ := simulator.ScenarioByID("database_overload")
	env := simulator.New(sc)
	inc := incident.New("inc-bad", sc.Alert, fixedNow())
	bad := llm.ScriptedLLM{Raw: `{"summary":"x","recommended_action":"rm_minus_rf_slash","recommended_service":"database","risk":"high"}`}
	runner := NewRunner(tools.NewRegistry(), bad, 2)
	res, err := runner.Run(context.Background(), env, inc, Options{Workers: 2, AutoApproveMedium: true, Now: fixedNow})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if res.Rejected == nil {
		t.Fatal("expected the hallucinated tool to be rejected")
	}
	if res.Incident.State != models.StateInvestigationFailed {
		t.Fatalf("rejected output should fail safe to INVESTIGATION_FAILED, got %s", res.Incident.State)
	}
	if env.Fixed() {
		t.Fatal("environment must be untouched after a rejected diagnosis")
	}
}

func TestEvidenceCollectedConcurrentlyIsDeterministic(t *testing.T) {
	sc, _ := simulator.ScenarioByID("cascading_failure")
	c := NewCollector(tools.NewRegistry(), 8)
	env1 := simulator.New(sc)
	env2 := simulator.New(sc)
	a, _ := c.Collect(context.Background(), env1, sc.Alert.Service, fixedNow)
	b, _ := c.Collect(context.Background(), env2, sc.Alert.Service, fixedNow)
	if len(a) != len(b) {
		t.Fatalf("evidence length differs across runs: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].ID != b[i].ID || a[i].Summary != b[i].Summary {
			t.Fatalf("evidence %d differs across runs despite concurrency", i)
		}
	}
}
