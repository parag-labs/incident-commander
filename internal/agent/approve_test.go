package agent

import (
	"context"
	"testing"

	"github.com/parag-labs/incident-commander/internal/agent/llm"
	"github.com/parag-labs/incident-commander/internal/incident"
	"github.com/parag-labs/incident-commander/internal/simulator"
	"github.com/parag-labs/incident-commander/internal/tools"
	"github.com/parag-labs/incident-commander/pkg/models"
)

func TestHumanApprovalExecutesHighRiskRollback(t *testing.T) {
	sc, _ := simulator.ScenarioByID("deployment_regression")
	env := simulator.New(sc)
	inc := incident.New("inc-rollback", sc.Alert, fixedNow())
	runner := NewRunner(tools.NewRegistry(), llm.MockLLM{}, 4)

	// First pass stops for approval.
	res, err := runner.Run(context.Background(), env, inc, Options{Workers: 4, AutoApproveMedium: true, Now: fixedNow})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if res.Incident.State != models.StateWaitingForApproval {
		t.Fatalf("want WAITING_FOR_APPROVAL, got %s", res.Incident.State)
	}
	if env.Fixed() {
		t.Fatal("nothing should have executed before approval")
	}

	// Human approves; now the HIGH rollback runs and the incident resolves.
	res2, err := runner.Approve(context.Background(), env, inc, Options{Now: fixedNow})
	if err != nil {
		t.Fatalf("approve error: %v", err)
	}
	if res2.Incident.State != models.StateResolved {
		t.Fatalf("after approval want RESOLVED, got %s", res2.Incident.State)
	}
	if !env.Fixed() {
		t.Fatal("environment should be fixed after approved rollback")
	}
}

func TestApproveRejectedUnlessWaiting(t *testing.T) {
	sc, _ := simulator.ScenarioByID("database_overload")
	env := simulator.New(sc)
	inc := incident.New("inc-x", sc.Alert, fixedNow())
	runner := NewRunner(tools.NewRegistry(), llm.MockLLM{}, 2)
	if _, err := runner.Approve(context.Background(), env, inc, Options{Now: fixedNow}); err != ErrNotAwaitingApproval {
		t.Fatalf("want ErrNotAwaitingApproval, got %v", err)
	}
}
