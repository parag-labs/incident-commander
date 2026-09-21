"""Typed domain contracts shared across the incident commander.

Nothing here does work; these are the nouns the deterministic core agrees on. The
enums are modelled as open string types (module-level constants) exactly like the Go
reference's ``type X string`` declarations, so an out-of-band value such as an unknown
safety level is representable and the policy engine's default branch stays reachable.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime
from typing import Any

# Service is one component of the simulated production environment.
Service = str
SERVICE_API: Service = "api"
SERVICE_PAYMENT: Service = "payment"
SERVICE_DATABASE: Service = "database"
SERVICE_CACHE: Service = "cache"
SERVICE_QUEUE: Service = "queue"

# AllServices is the fleet the simulator models, in a stable order.
ALL_SERVICES: list[Service] = [
    SERVICE_API,
    SERVICE_PAYMENT,
    SERVICE_DATABASE,
    SERVICE_CACHE,
    SERVICE_QUEUE,
]

# Severity is how bad an alert looks on arrival.
Severity = str
SEVERITY_INFO: Severity = "info"
SEVERITY_WARNING: Severity = "warning"
SEVERITY_CRITICAL: Severity = "critical"

# State is a node in the incident state machine.
State = str
STATE_NEW: State = "NEW"
STATE_TRIAGING: State = "TRIAGING"
STATE_INVESTIGATING: State = "INVESTIGATING"
STATE_ROOT_CAUSE_IDENTIFIED: State = "ROOT_CAUSE_IDENTIFIED"
STATE_REMEDIATION_PROPOSED: State = "REMEDIATION_PROPOSED"
STATE_WAITING_FOR_APPROVAL: State = "WAITING_FOR_APPROVAL"
STATE_REMEDIATING: State = "REMEDIATING"
STATE_VERIFYING: State = "VERIFYING"
STATE_RESOLVED: State = "RESOLVED"
STATE_INVESTIGATION_FAILED: State = "INVESTIGATION_FAILED"
STATE_REMEDIATION_FAILED: State = "REMEDIATION_FAILED"
STATE_VERIFICATION_FAILED: State = "VERIFICATION_FAILED"
STATE_ESCALATED: State = "ESCALATED"

# SafetyLevel classifies how dangerous a tool is.
SafetyLevel = str
SAFETY_LOW: SafetyLevel = "low"
SAFETY_MEDIUM: SafetyLevel = "medium"
SAFETY_HIGH: SafetyLevel = "high"

# Risk is the qualitative risk the agent assigns to a recommended action.
Risk = str
RISK_LOW: Risk = "low"
RISK_MEDIUM: Risk = "medium"
RISK_HIGH: Risk = "high"


@dataclass
class Alert:
    """The signal that opens an incident."""

    service: Service = ""
    metric: str = ""
    value: float = 0.0
    severity: Severity = ""
    message: str = ""
    id: str = ""
    fired_at: datetime | None = None


@dataclass
class Evidence:
    """One grounded fact a tool produced."""

    id: str = ""
    tool: str = ""
    service: Service = ""
    summary: str = ""
    detail: dict[str, Any] | None = None


@dataclass
class Hypothesis:
    """A candidate root cause with a confidence and the evidence behind it."""

    cause: str = ""
    confidence: float = 0.0
    evidence_ids: list[str] = field(default_factory=list)


@dataclass
class ToolCall:
    """Records that a tool ran, for the audit trail."""

    tool: str = ""
    service: Service = ""
    safety: SafetyLevel = ""
    ok: bool = False
    error: str = ""
    duration_ms: int = 0
    args: dict[str, Any] | None = None
    started_at: datetime | None = None


@dataclass
class Remediation:
    """A proposed corrective action targeting one service."""

    action: str = ""
    service: Service = ""
    safety: SafetyLevel = ""
    reason: str = ""


@dataclass
class RiskAssessment:
    """The deterministic policy engine's verdict on a remediation."""

    safety: SafetyLevel = ""
    auto_approvable: bool = False
    needs_approval: bool = False
    reason: str = ""


@dataclass
class Diagnosis:
    """The validated, structured output of the AI investigator."""

    summary: str = ""
    hypotheses: list[Hypothesis] = field(default_factory=list)
    recommended_action: str = ""
    recommended_service: Service = ""
    risk: Risk = ""
    needs_human_approval: bool = False

    def top(self) -> Hypothesis | None:
        """Return the highest-confidence hypothesis, or None if there are none.

        Ties keep the first hypothesis seen, matching the reference's strict ``>``.
        """
        best: Hypothesis | None = None
        best_conf = -1.0
        for h in self.hypotheses:
            if h.confidence > best_conf:
                best = h
                best_conf = h.confidence
        return best


@dataclass
class VerificationResult:
    """Records whether an executed remediation actually improved things."""

    improved: bool = False
    before_err_pct: float = 0.0
    after_err_pct: float = 0.0
    before_p99_ms: float = 0.0
    after_p99_ms: float = 0.0
    note: str = ""


@dataclass
class Transition:
    """One recorded step of the state machine, for the audit trail."""

    frm: State = ""
    to: State = ""
    reason: str = ""
    at: datetime | None = None


@dataclass
class IncidentReport:
    """The evidence-backed narrative produced at the end."""

    incident_id: str = ""
    title: str = ""
    state: State = ""
    root_cause: str = ""
    diagnosis: Diagnosis | None = None
    remediation: Remediation | None = None
    verification: VerificationResult | None = None
    evidence: list[Evidence] = field(default_factory=list)
    tool_calls: list[ToolCall] = field(default_factory=list)
    generated_at: datetime | None = None


@dataclass
class Incident:
    """The top-level aggregate the state machine advances."""

    id: str = ""
    title: str = ""
    state: State = ""
    alert: Alert = field(default_factory=Alert)
    evidence: list[Evidence] = field(default_factory=list)
    tool_calls: list[ToolCall] = field(default_factory=list)
    diagnosis: Diagnosis | None = None
    remediation: Remediation | None = None
    risk: RiskAssessment | None = None
    verification: VerificationResult | None = None
    created_at: datetime | None = None
    updated_at: datetime | None = None
    history: list[Transition] = field(default_factory=list)
