package eval

import "testing"

func TestEvaluationMeetsQualityBar(t *testing.T) {
	r := Run()
	if r.Scenarios != 10 {
		t.Fatalf("want 10 scenarios, got %d", r.Scenarios)
	}
	// The mock reasoner is deterministic, so these are exact, not flaky thresholds.
	if r.RootCauseAccuracy() < 1.0 {
		t.Errorf("root cause accuracy dropped below 100%%: %.0f%% (%+v)", 100*r.RootCauseAccuracy(), r.Cases)
	}
	if r.EvidenceGrounded != r.Scenarios {
		t.Errorf("every diagnosis should be grounded, got %d/%d", r.EvidenceGrounded, r.Scenarios)
	}
	if r.UnsafeActions != 0 {
		t.Errorf("there must be zero unsafe actions, got %d", r.UnsafeActions)
	}
	if r.RemediationSuccess() < 1.0 {
		t.Errorf("remediation success dropped below 100%%: %.0f%%", 100*r.RemediationSuccess())
	}
}
