// Package policy is the deterministic risk gate between the AI's recommendation and
// any action that touches the environment. The agent can recommend anything; nothing
// runs until this engine has classified it and decided whether it may run
// automatically. This is the hard boundary that keeps free-form model output from ever
// executing an infrastructure command on its own.
package policy

import (
	"fmt"

	"github.com/parag-labs/incident-commander/pkg/models"
)

// Config controls how permissive the gate is.
type Config struct {
	// AutoApproveMedium lets MEDIUM-safety remediations run without a human. LOW always
	// may; HIGH never may, regardless of this flag.
	AutoApproveMedium bool
}

// Assess classifies a remediation and decides whether it may run automatically. The
// safety level is the tool's declared level (looked up by the caller), not anything the
// model asserts.
func Assess(rem models.Remediation, safety models.SafetyLevel, cfg Config) models.RiskAssessment {
	ra := models.RiskAssessment{Safety: safety}
	switch safety {
	case models.SafetyLow:
		ra.AutoApprovable = true
		ra.Reason = "read-only / low-risk action; safe to run automatically"
	case models.SafetyMedium:
		if cfg.AutoApproveMedium {
			ra.AutoApprovable = true
			ra.Reason = "medium-risk action; auto-approval is enabled"
		} else {
			ra.NeedsApproval = true
			ra.Reason = "medium-risk action; requires human approval (auto-approval disabled)"
		}
	case models.SafetyHigh:
		ra.NeedsApproval = true
		ra.Reason = "high-risk action; never runs without explicit human approval"
	default:
		ra.NeedsApproval = true
		ra.Reason = fmt.Sprintf("unknown safety level %q; defaulting to human approval", safety)
	}
	return ra
}
