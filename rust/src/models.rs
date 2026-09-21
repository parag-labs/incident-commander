//! Typed domain contracts shared across the incident commander.
//!
//! The enum-like types are modelled as open string constants exactly like the Go
//! reference's `type X string` declarations, so an out-of-band value such as an unknown
//! safety level is representable and the policy engine's default branch stays reachable.

/// A component of the simulated production environment.
pub const SERVICE_API: &str = "api";
/// The payment service.
pub const SERVICE_PAYMENT: &str = "payment";
/// The database service.
pub const SERVICE_DATABASE: &str = "database";
/// The cache service.
pub const SERVICE_CACHE: &str = "cache";
/// The queue service.
pub const SERVICE_QUEUE: &str = "queue";

/// The fleet the simulator models, in a stable order.
pub const ALL_SERVICES: [&str; 5] = [
    SERVICE_API,
    SERVICE_PAYMENT,
    SERVICE_DATABASE,
    SERVICE_CACHE,
    SERVICE_QUEUE,
];

/// An informational alert severity.
pub const SEVERITY_INFO: &str = "info";
/// A warning alert severity.
pub const SEVERITY_WARNING: &str = "warning";
/// A critical alert severity.
pub const SEVERITY_CRITICAL: &str = "critical";

/// The initial state of every incident.
pub const STATE_NEW: &str = "NEW";
/// Triage is under way.
pub const STATE_TRIAGING: &str = "TRIAGING";
/// Investigation is under way.
pub const STATE_INVESTIGATING: &str = "INVESTIGATING";
/// A root cause has been identified.
pub const STATE_ROOT_CAUSE_IDENTIFIED: &str = "ROOT_CAUSE_IDENTIFIED";
/// A remediation has been proposed.
pub const STATE_REMEDIATION_PROPOSED: &str = "REMEDIATION_PROPOSED";
/// The remediation is waiting for human approval.
pub const STATE_WAITING_FOR_APPROVAL: &str = "WAITING_FOR_APPROVAL";
/// The remediation is executing.
pub const STATE_REMEDIATING: &str = "REMEDIATING";
/// The remediation is being verified.
pub const STATE_VERIFYING: &str = "VERIFYING";
/// The incident is resolved (terminal).
pub const STATE_RESOLVED: &str = "RESOLVED";
/// Investigation failed (terminal).
pub const STATE_INVESTIGATION_FAILED: &str = "INVESTIGATION_FAILED";
/// Remediation failed (terminal).
pub const STATE_REMEDIATION_FAILED: &str = "REMEDIATION_FAILED";
/// Verification failed (terminal).
pub const STATE_VERIFICATION_FAILED: &str = "VERIFICATION_FAILED";
/// The incident was escalated to a human (terminal).
pub const STATE_ESCALATED: &str = "ESCALATED";

/// A read-only / low-risk safety level.
pub const SAFETY_LOW: &str = "low";
/// A medium-risk safety level.
pub const SAFETY_MEDIUM: &str = "medium";
/// A high-risk safety level.
pub const SAFETY_HIGH: &str = "high";

/// Low qualitative risk.
pub const RISK_LOW: &str = "low";
/// Medium qualitative risk.
pub const RISK_MEDIUM: &str = "medium";
/// High qualitative risk.
pub const RISK_HIGH: &str = "high";

/// The signal that opens an incident.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Alert {
    /// The service the alert fired on.
    pub service: String,
    /// The metric that breached.
    pub metric: String,
    /// The observed value.
    pub value: f64,
    /// The severity as reported.
    pub severity: String,
    /// A human-readable message.
    pub message: String,
    /// An optional identifier.
    pub id: String,
    /// When the alert fired (unix nanoseconds; never inspected by the core).
    pub fired_at: i64,
}

/// One grounded fact a tool produced.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Evidence {
    /// The evidence identifier cited by hypotheses.
    pub id: String,
    /// The tool that produced it.
    pub tool: String,
    /// The service it concerns.
    pub service: String,
    /// A human-readable summary.
    pub summary: String,
    /// Optional structured detail (unused by the core).
    pub detail: Option<Vec<(String, String)>>,
}

/// A candidate root cause with a confidence and the evidence behind it.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Hypothesis {
    /// The proposed cause.
    pub cause: String,
    /// The confidence in `[0, 1]`.
    pub confidence: f64,
    /// The evidence identifiers cited in support.
    pub evidence_ids: Vec<String>,
}

/// Records that a tool ran, for the audit trail.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct ToolCall {
    /// The tool name.
    pub tool: String,
    /// The service it targeted.
    pub service: String,
    /// The tool's declared safety level.
    pub safety: String,
    /// Whether it completed successfully.
    pub ok: bool,
    /// Any error message.
    pub error: String,
    /// How long it took, in milliseconds.
    pub duration_ms: i64,
    /// Optional arguments (unused by the core).
    pub args: Option<Vec<(String, String)>>,
    /// When it started (unix nanoseconds; never inspected).
    pub started_at: i64,
}

/// A proposed corrective action targeting one service.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Remediation {
    /// The action name (matched against tool calls).
    pub action: String,
    /// The service it targets.
    pub service: String,
    /// The declared safety level.
    pub safety: String,
    /// Why it was proposed.
    pub reason: String,
}

/// The deterministic policy engine's verdict on a remediation.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct RiskAssessment {
    /// The safety level assessed.
    pub safety: String,
    /// Whether the action may run automatically.
    pub auto_approvable: bool,
    /// Whether the action requires human approval.
    pub needs_approval: bool,
    /// The human-readable justification.
    pub reason: String,
}

/// The validated, structured output of the AI investigator.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Diagnosis {
    /// A narrative summary.
    pub summary: String,
    /// The ranked hypotheses.
    pub hypotheses: Vec<Hypothesis>,
    /// The recommended action.
    pub recommended_action: String,
    /// The recommended service.
    pub recommended_service: String,
    /// The qualitative risk.
    pub risk: String,
    /// Whether human approval is requested.
    pub needs_human_approval: bool,
}

impl Diagnosis {
    /// Return the highest-confidence hypothesis, or `None` if there are none.
    ///
    /// Ties keep the first hypothesis seen, matching the reference's strict `>`.
    pub fn top(&self) -> Option<&Hypothesis> {
        let mut best: Option<&Hypothesis> = None;
        let mut best_conf = -1.0_f64;
        for h in &self.hypotheses {
            if h.confidence > best_conf {
                best = Some(h);
                best_conf = h.confidence;
            }
        }
        best
    }
}

/// Records whether an executed remediation actually improved things.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct VerificationResult {
    /// Whether the situation improved.
    pub improved: bool,
    /// Error percentage before.
    pub before_err_pct: f64,
    /// Error percentage after.
    pub after_err_pct: f64,
    /// p99 latency before, in milliseconds.
    pub before_p99_ms: f64,
    /// p99 latency after, in milliseconds.
    pub after_p99_ms: f64,
    /// A human-readable note.
    pub note: String,
}

/// One recorded step of the state machine, for the audit trail.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Transition {
    /// The state moved from (empty for the opening entry).
    pub from: String,
    /// The state moved to.
    pub to: String,
    /// Why the move happened.
    pub reason: String,
    /// When it happened (unix nanoseconds; never inspected).
    pub at: i64,
}

/// The evidence-backed narrative produced at the end.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct IncidentReport {
    /// The incident identifier.
    pub incident_id: String,
    /// The incident title.
    pub title: String,
    /// The final state.
    pub state: String,
    /// The root cause narrative.
    pub root_cause: String,
    /// The diagnosis, if any.
    pub diagnosis: Option<Diagnosis>,
    /// The remediation, if any.
    pub remediation: Option<Remediation>,
    /// The verification result, if any.
    pub verification: Option<VerificationResult>,
    /// The collected evidence.
    pub evidence: Vec<Evidence>,
    /// The recorded tool calls.
    pub tool_calls: Vec<ToolCall>,
    /// When the report was generated (unix nanoseconds).
    pub generated_at: i64,
}

/// The top-level aggregate the state machine advances.
#[derive(Debug, Clone, PartialEq, Default)]
pub struct Incident {
    /// The incident identifier.
    pub id: String,
    /// The incident title.
    pub title: String,
    /// The current state.
    pub state: String,
    /// The alert that opened it.
    pub alert: Alert,
    /// The collected evidence.
    pub evidence: Vec<Evidence>,
    /// The recorded tool calls.
    pub tool_calls: Vec<ToolCall>,
    /// The diagnosis, if any.
    pub diagnosis: Option<Diagnosis>,
    /// The remediation, if any.
    pub remediation: Option<Remediation>,
    /// The policy verdict, if any.
    pub risk: Option<RiskAssessment>,
    /// The verification result, if any.
    pub verification: Option<VerificationResult>,
    /// When it was created (unix nanoseconds).
    pub created_at: i64,
    /// When it was last updated (unix nanoseconds).
    pub updated_at: i64,
    /// The full transition history.
    pub history: Vec<Transition>,
}
