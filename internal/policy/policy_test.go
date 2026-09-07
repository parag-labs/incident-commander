package policy

import (
	"testing"

	"github.com/parag-labs/incident-commander/pkg/models"
)

func TestLowAlwaysAutoApprovable(t *testing.T) {
	ra := Assess(models.Remediation{Action: "query_metrics"}, models.SafetyLow, Config{})
	if !ra.AutoApprovable || ra.NeedsApproval {
		t.Fatalf("LOW should be auto-approvable, got %+v", ra)
	}
}

func TestMediumDependsOnConfig(t *testing.T) {
	rem := models.Remediation{Action: "restart_service"}

	off := Assess(rem, models.SafetyMedium, Config{AutoApproveMedium: false})
	if off.AutoApprovable || !off.NeedsApproval {
		t.Fatalf("MEDIUM without auto-approval should need approval, got %+v", off)
	}

	on := Assess(rem, models.SafetyMedium, Config{AutoApproveMedium: true})
	if !on.AutoApprovable || on.NeedsApproval {
		t.Fatalf("MEDIUM with auto-approval should be auto-approvable, got %+v", on)
	}
}

func TestHighNeverAutoApprovable(t *testing.T) {
	for _, cfg := range []Config{{AutoApproveMedium: false}, {AutoApproveMedium: true}} {
		ra := Assess(models.Remediation{Action: "rollback_deployment"}, models.SafetyHigh, cfg)
		if ra.AutoApprovable || !ra.NeedsApproval {
			t.Fatalf("HIGH must never be auto-approvable (cfg=%+v), got %+v", cfg, ra)
		}
	}
}

func TestUnknownSafetyDefaultsToApproval(t *testing.T) {
	ra := Assess(models.Remediation{Action: "???"}, models.SafetyLevel("weird"), Config{AutoApproveMedium: true})
	if ra.AutoApprovable {
		t.Fatalf("unknown safety must not be auto-approvable, got %+v", ra)
	}
}
