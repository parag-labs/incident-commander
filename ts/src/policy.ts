// The deterministic risk gate between the AI's recommendation and any action, ported from
// the Go reference.
//
// The agent can recommend anything; nothing runs until this engine has classified it and
// decided whether it may run automatically. This is the hard boundary that keeps free-form
// model output from ever executing an infrastructure command on its own.

import { Remediation, RiskAssessment, Safety, SafetyLevel } from "./models.js";

/**
 * Controls how permissive the gate is. `autoApproveMedium` lets MEDIUM-safety remediations
 * run without a human. LOW always may; HIGH never may, regardless of this flag.
 */
export interface Config {
  autoApproveMedium: boolean;
}

/**
 * Classify a remediation and decide whether it may run automatically. The safety level is
 * the tool's declared level (looked up by the caller), not anything the model asserts.
 * `_rem` is part of the audit contract but does not change the verdict, matching the
 * reference.
 */
export function assess(_rem: Remediation, safety: SafetyLevel, cfg: Config): RiskAssessment {
  const ra = new RiskAssessment({ safety });
  if (safety === Safety.Low) {
    ra.autoApprovable = true;
    ra.reason = "read-only / low-risk action; safe to run automatically";
  } else if (safety === Safety.Medium) {
    if (cfg.autoApproveMedium) {
      ra.autoApprovable = true;
      ra.reason = "medium-risk action; auto-approval is enabled";
    } else {
      ra.needsApproval = true;
      ra.reason = "medium-risk action; requires human approval (auto-approval disabled)";
    }
  } else if (safety === Safety.High) {
    ra.needsApproval = true;
    ra.reason = "high-risk action; never runs without explicit human approval";
  } else {
    ra.needsApproval = true;
    ra.reason = `unknown safety level "${safety}"; defaulting to human approval`;
  }
  return ra;
}
