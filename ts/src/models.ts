// Typed domain contracts shared across the incident commander, ported from the Go
// reference. Nothing here does work; these are the nouns the deterministic core agrees on.
// The enum-like types are modelled as open string aliases exactly like the Go reference's
// `type X string` declarations, so an out-of-band value such as an unknown safety level is
// representable and the policy engine's default branch stays reachable.

/** One component of the simulated production environment. */
export type Service = string;

/** Named services, mirroring the Go constants. */
export const Services = {
  API: "api",
  Payment: "payment",
  Database: "database",
  Cache: "cache",
  Queue: "queue",
} as const;

/** The fleet the simulator models, in a stable order. */
export const ALL_SERVICES: readonly Service[] = [
  Services.API,
  Services.Payment,
  Services.Database,
  Services.Cache,
  Services.Queue,
];

/** How bad an alert looks on arrival. */
export type Severity = string;

/** Named severities. */
export const Severities = {
  Info: "info",
  Warning: "warning",
  Critical: "critical",
} as const;

/** A node in the incident state machine. */
export type State = string;

/** Named states. */
export const States = {
  New: "NEW",
  Triaging: "TRIAGING",
  Investigating: "INVESTIGATING",
  RootCauseIdentified: "ROOT_CAUSE_IDENTIFIED",
  RemediationProposed: "REMEDIATION_PROPOSED",
  WaitingForApproval: "WAITING_FOR_APPROVAL",
  Remediating: "REMEDIATING",
  Verifying: "VERIFYING",
  Resolved: "RESOLVED",
  InvestigationFailed: "INVESTIGATION_FAILED",
  RemediationFailed: "REMEDIATION_FAILED",
  VerificationFailed: "VERIFICATION_FAILED",
  Escalated: "ESCALATED",
} as const;

/** Classifies how dangerous a tool is. */
export type SafetyLevel = string;

/** Named safety levels. */
export const Safety = {
  Low: "low",
  Medium: "medium",
  High: "high",
} as const;

/** The qualitative risk the agent assigns to a recommended action. */
export type Risk = string;

/** Named risk levels. */
export const Risks = {
  Low: "low",
  Medium: "medium",
  High: "high",
} as const;

/** The signal that opens an incident. */
export class Alert {
  id = "";
  service: Service = "";
  metric = "";
  value = 0;
  severity: Severity = "";
  message = "";
  firedAt: Date | null = null;

  constructor(init?: Partial<Alert>) {
    Object.assign(this, init);
  }
}

/** One grounded fact a tool produced. */
export class Evidence {
  id = "";
  tool = "";
  service: Service = "";
  summary = "";
  detail: Record<string, unknown> | null = null;

  constructor(init?: Partial<Evidence>) {
    Object.assign(this, init);
  }
}

/** A candidate root cause with a confidence and the evidence behind it. */
export class Hypothesis {
  cause = "";
  confidence = 0;
  evidenceIds: string[] = [];

  constructor(init?: Partial<Hypothesis>) {
    Object.assign(this, init);
  }
}

/** Records that a tool ran, for the audit trail. */
export class ToolCall {
  tool = "";
  service: Service = "";
  safety: SafetyLevel = "";
  ok = false;
  error = "";
  durationMs = 0;
  args: Record<string, unknown> | null = null;
  startedAt: Date | null = null;

  constructor(init?: Partial<ToolCall>) {
    Object.assign(this, init);
  }
}

/** A proposed corrective action targeting one service. */
export class Remediation {
  action = "";
  service: Service = "";
  safety: SafetyLevel = "";
  reason = "";

  constructor(init?: Partial<Remediation>) {
    Object.assign(this, init);
  }
}

/** The deterministic policy engine's verdict on a remediation. */
export class RiskAssessment {
  safety: SafetyLevel = "";
  autoApprovable = false;
  needsApproval = false;
  reason = "";

  constructor(init?: Partial<RiskAssessment>) {
    Object.assign(this, init);
  }
}

/** The validated, structured output of the AI investigator. */
export class Diagnosis {
  summary = "";
  hypotheses: Hypothesis[] = [];
  recommendedAction = "";
  recommendedService: Service = "";
  risk: Risk = "";
  needsHumanApproval = false;

  constructor(init?: Partial<Diagnosis>) {
    Object.assign(this, init);
  }

  /**
   * Return the highest-confidence hypothesis, or `undefined` if there are none. Ties keep
   * the first hypothesis seen, matching the reference's strict `>`.
   */
  top(): Hypothesis | undefined {
    let best: Hypothesis | undefined;
    let bestConf = -1.0;
    for (const h of this.hypotheses) {
      if (h.confidence > bestConf) {
        best = h;
        bestConf = h.confidence;
      }
    }
    return best;
  }
}

/** Records whether an executed remediation actually improved things. */
export class VerificationResult {
  improved = false;
  beforeErrPct = 0;
  afterErrPct = 0;
  beforeP99Ms = 0;
  afterP99Ms = 0;
  note = "";

  constructor(init?: Partial<VerificationResult>) {
    Object.assign(this, init);
  }
}

/** One recorded step of the state machine, for the audit trail. */
export class Transition {
  from: State = "";
  to: State = "";
  reason = "";
  at: Date | null = null;

  constructor(init?: Partial<Transition>) {
    Object.assign(this, init);
  }
}

/** The evidence-backed narrative produced at the end. */
export class IncidentReport {
  incidentId = "";
  title = "";
  state: State = "";
  rootCause = "";
  diagnosis: Diagnosis | null = null;
  remediation: Remediation | null = null;
  verification: VerificationResult | null = null;
  evidence: Evidence[] = [];
  toolCalls: ToolCall[] = [];
  generatedAt: Date | null = null;

  constructor(init?: Partial<IncidentReport>) {
    Object.assign(this, init);
  }
}

/** The top-level aggregate the state machine advances. */
export class Incident {
  id = "";
  title = "";
  state: State = "";
  alert: Alert = new Alert();
  evidence: Evidence[] = [];
  toolCalls: ToolCall[] = [];
  diagnosis: Diagnosis | null = null;
  remediation: Remediation | null = null;
  risk: RiskAssessment | null = null;
  verification: VerificationResult | null = null;
  createdAt: Date | null = null;
  updatedAt: Date | null = null;
  history: Transition[] = [];

  constructor(init?: Partial<Incident>) {
    Object.assign(this, init);
  }
}
