package incident

import (
	"testing"
	"time"

	"github.com/parag-labs/incident-commander/pkg/models"
)

func newTestIncident() *models.Incident {
	return New("inc-1", models.Alert{
		Service:  models.ServiceDatabase,
		Metric:   "error_rate",
		Severity: models.SeverityCritical,
	}, time.Unix(0, 0))
}

func TestNewStartsInNew(t *testing.T) {
	inc := newTestIncident()
	if inc.State != models.StateNew {
		t.Fatalf("want NEW, got %s", inc.State)
	}
	if len(inc.History) != 1 || inc.History[0].To != models.StateNew {
		t.Fatalf("expected an opening history entry, got %+v", inc.History)
	}
}

func TestHappyPathWalksToResolved(t *testing.T) {
	inc := newTestIncident()
	path := []models.State{
		models.StateTriaging,
		models.StateInvestigating,
		models.StateRootCauseIdentified,
		models.StateRemediationProposed,
		models.StateWaitingForApproval,
		models.StateRemediating,
		models.StateVerifying,
		models.StateResolved,
	}
	for _, to := range path {
		if err := Transition(inc, to, "step", time.Unix(0, 0)); err != nil {
			t.Fatalf("legal transition to %s rejected: %v", to, err)
		}
	}
	if inc.State != models.StateResolved {
		t.Fatalf("want RESOLVED, got %s", inc.State)
	}
	// opening entry + 8 transitions
	if len(inc.History) != 9 {
		t.Fatalf("want 9 history entries, got %d", len(inc.History))
	}
}

func TestIllegalTransitionIsRejectedAndDoesNotMutate(t *testing.T) {
	inc := newTestIncident()
	// NEW cannot jump straight to REMEDIATING.
	err := Transition(inc, models.StateRemediating, "skip", time.Unix(0, 0))
	if err == nil {
		t.Fatal("expected illegal transition to be rejected")
	}
	if _, ok := err.(ErrIllegalTransition); !ok {
		t.Fatalf("want ErrIllegalTransition, got %T", err)
	}
	if inc.State != models.StateNew {
		t.Fatalf("state must be unchanged after a rejected move, got %s", inc.State)
	}
	if len(inc.History) != 1 {
		t.Fatalf("history must be unchanged after a rejected move, got %d entries", len(inc.History))
	}
}

func TestTerminalStatesHaveNoExit(t *testing.T) {
	for _, s := range []models.State{
		models.StateResolved, models.StateEscalated,
		models.StateInvestigationFailed, models.StateRemediationFailed, models.StateVerificationFailed,
	} {
		if !IsTerminal(s) {
			t.Errorf("%s should be terminal", s)
		}
		inc := newTestIncident()
		inc.State = s
		if err := Transition(inc, models.StateResolved, "x", time.Unix(0, 0)); err == nil {
			t.Errorf("terminal state %s allowed an outgoing transition", s)
		}
	}
}

func TestInvestigationCanFail(t *testing.T) {
	inc := newTestIncident()
	mustTransition(t, inc, models.StateTriaging)
	mustTransition(t, inc, models.StateInvestigating)
	if err := Transition(inc, models.StateInvestigationFailed, "no evidence", time.Unix(0, 0)); err != nil {
		t.Fatalf("investigation should be allowed to fail: %v", err)
	}
	if !IsTerminal(inc.State) {
		t.Fatal("INVESTIGATION_FAILED should be terminal")
	}
}

func TestReadOnlyResolutionSkipsRemediation(t *testing.T) {
	// A read-only incident can resolve straight from ROOT_CAUSE_IDENTIFIED.
	inc := newTestIncident()
	mustTransition(t, inc, models.StateTriaging)
	mustTransition(t, inc, models.StateInvestigating)
	mustTransition(t, inc, models.StateRootCauseIdentified)
	if err := Transition(inc, models.StateResolved, "informational", time.Unix(0, 0)); err != nil {
		t.Fatalf("root cause -> resolved should be legal: %v", err)
	}
}

func mustTransition(t *testing.T, inc *models.Incident, to models.State) {
	t.Helper()
	if err := Transition(inc, to, "step", time.Unix(0, 0)); err != nil {
		t.Fatalf("transition to %s failed: %v", to, err)
	}
}
