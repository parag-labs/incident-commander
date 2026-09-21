//! The deterministic risk gate between the AI's recommendation and any action.
//!
//! The agent can recommend anything; nothing runs until this engine has classified it
//! and decided whether it may run automatically.

use crate::models::{Remediation, RiskAssessment, SAFETY_HIGH, SAFETY_LOW, SAFETY_MEDIUM};

/// Controls how permissive the gate is.
#[derive(Debug, Clone, Copy, Default)]
pub struct Config {
    /// Lets MEDIUM-safety remediations run without a human. LOW always may; HIGH never
    /// may, regardless of this flag.
    pub auto_approve_medium: bool,
}

/// Classify a remediation and decide whether it may run automatically.
///
/// The safety level is the tool's declared level (looked up by the caller), not anything
/// the model asserts. `rem` is part of the audit contract but does not change the
/// verdict, matching the reference.
pub fn assess(_rem: &Remediation, safety: &str, cfg: Config) -> RiskAssessment {
    let mut ra = RiskAssessment {
        safety: safety.to_string(),
        ..Default::default()
    };
    if safety == SAFETY_LOW {
        ra.auto_approvable = true;
        ra.reason = "read-only / low-risk action; safe to run automatically".to_string();
    } else if safety == SAFETY_MEDIUM {
        if cfg.auto_approve_medium {
            ra.auto_approvable = true;
            ra.reason = "medium-risk action; auto-approval is enabled".to_string();
        } else {
            ra.needs_approval = true;
            ra.reason =
                "medium-risk action; requires human approval (auto-approval disabled)".to_string();
        }
    } else if safety == SAFETY_HIGH {
        ra.needs_approval = true;
        ra.reason = "high-risk action; never runs without explicit human approval".to_string();
    } else {
        ra.needs_approval = true;
        ra.reason = format!("unknown safety level \"{safety}\"; defaulting to human approval");
    }
    ra
}
