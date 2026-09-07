// Package models holds the typed domain contracts shared across the incident
// commander. Nothing here does work; these are the nouns the deterministic core and
// the AI layer both agree on, so no part of the system has to pass untyped maps
// around. Enums are plain strings so they serialize to readable JSON.
package models

import "time"

// Service is one component of the simulated production environment.
type Service string

const (
	ServiceAPI      Service = "api"
	ServicePayment  Service = "payment"
	ServiceDatabase Service = "database"
	ServiceCache    Service = "cache"
	ServiceQueue    Service = "queue"
)

// AllServices is the fleet the simulator models, in a stable order.
var AllServices = []Service{ServiceAPI, ServicePayment, ServiceDatabase, ServiceCache, ServiceQueue}

// Severity is how bad an alert looks on arrival.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// State is a node in the incident state machine. The machine is deterministic and
// the LLM never sets it directly - only the orchestrator advances it through
// validated transitions.
type State string

const (
	StateNew                 State = "NEW"
	StateTriaging            State = "TRIAGING"
	StateInvestigating       State = "INVESTIGATING"
	StateRootCauseIdentified State = "ROOT_CAUSE_IDENTIFIED"
	StateRemediationProposed State = "REMEDIATION_PROPOSED"
	StateWaitingForApproval  State = "WAITING_FOR_APPROVAL"
	StateRemediating         State = "REMEDIATING"
	StateVerifying           State = "VERIFYING"
	StateResolved            State = "RESOLVED"

	StateInvestigationFailed State = "INVESTIGATION_FAILED"
	StateRemediationFailed   State = "REMEDIATION_FAILED"
	StateVerificationFailed  State = "VERIFICATION_FAILED"
	StateEscalated           State = "ESCALATED"
)

// SafetyLevel classifies how dangerous a tool is. It decides whether the deterministic
// policy engine will let the AI's recommendation run automatically.
type SafetyLevel string

const (
	// SafetyLow is read-only; always safe to run automatically.
	SafetyLow SafetyLevel = "low"
	// SafetyMedium is potentially disruptive; runs only if auto-approval is enabled.
	SafetyMedium SafetyLevel = "medium"
	// SafetyHigh is dangerous; never runs without a human.
	SafetyHigh SafetyLevel = "high"
)

// Risk is the qualitative risk the agent assigns to a recommended action.
type Risk string

const (
	RiskLow    Risk = "low"
	RiskMedium Risk = "medium"
	RiskHigh   Risk = "high"
)

// Alert is the signal that opens an incident.
type Alert struct {
	ID       string    `json:"id"`
	Service  Service   `json:"service"`
	Metric   string    `json:"metric"`
	Value    float64   `json:"value"`
	Severity Severity  `json:"severity"`
	Message  string    `json:"message"`
	FiredAt  time.Time `json:"fired_at"`
}

// Evidence is one grounded fact a tool produced. Every hypothesis the AI makes must
// cite Evidence IDs that actually exist - that is how hallucination is caught.
type Evidence struct {
	ID      string  `json:"id"`
	Tool    string  `json:"tool"`
	Service Service `json:"service"`
	Summary string  `json:"summary"`
	// Detail carries the structured tool output for the record.
	Detail map[string]any `json:"detail,omitempty"`
}

// Hypothesis is a candidate root cause with a confidence and the evidence behind it.
type Hypothesis struct {
	Cause       string   `json:"cause"`
	Confidence  float64  `json:"confidence"`
	EvidenceIDs []string `json:"evidence_ids"`
}

// ToolCall records that a tool ran, for the audit trail.
type ToolCall struct {
	Tool       string         `json:"tool"`
	Args       map[string]any `json:"args,omitempty"`
	Service    Service        `json:"service,omitempty"`
	Safety     SafetyLevel    `json:"safety"`
	OK         bool           `json:"ok"`
	Error      string         `json:"error,omitempty"`
	StartedAt  time.Time      `json:"started_at"`
	DurationMS int64          `json:"duration_ms"`
}

// Remediation is a proposed corrective action targeting one service.
type Remediation struct {
	Action  string      `json:"action"`
	Service Service     `json:"service"`
	Safety  SafetyLevel `json:"safety"`
	Reason  string      `json:"reason"`
}

// RiskAssessment is the deterministic policy engine's verdict on a remediation.
type RiskAssessment struct {
	Safety         SafetyLevel `json:"safety"`
	AutoApprovable bool        `json:"auto_approvable"`
	NeedsApproval  bool        `json:"needs_approval"`
	Reason         string      `json:"reason"`
}

// Diagnosis is the validated, structured output of the AI investigator. It is what the
// orchestrator acts on - never free-form model text.
type Diagnosis struct {
	Summary            string       `json:"summary"`
	Hypotheses         []Hypothesis `json:"hypotheses"`
	RecommendedAction  string       `json:"recommended_action"`
	RecommendedService Service      `json:"recommended_service"`
	Risk               Risk         `json:"risk"`
	NeedsHumanApproval bool         `json:"needs_human_approval"`
}

// Top returns the highest-confidence hypothesis, or false if there are none.
func (d Diagnosis) Top() (Hypothesis, bool) {
	best := Hypothesis{Confidence: -1}
	found := false
	for _, h := range d.Hypotheses {
		if h.Confidence > best.Confidence {
			best, found = h, true
		}
	}
	return best, found
}

// VerificationResult records whether an executed remediation actually improved things.
type VerificationResult struct {
	Improved     bool    `json:"improved"`
	BeforeErrPct float64 `json:"before_error_rate"`
	AfterErrPct  float64 `json:"after_error_rate"`
	BeforeP99MS  float64 `json:"before_latency_p99_ms"`
	AfterP99MS   float64 `json:"after_latency_p99_ms"`
	Note         string  `json:"note"`
}

// IncidentReport is the evidence-backed narrative produced at the end.
type IncidentReport struct {
	IncidentID   string              `json:"incident_id"`
	Title        string              `json:"title"`
	State        State               `json:"state"`
	RootCause    string              `json:"root_cause"`
	Diagnosis    Diagnosis           `json:"diagnosis"`
	Remediation  *Remediation        `json:"remediation,omitempty"`
	Verification *VerificationResult `json:"verification,omitempty"`
	Evidence     []Evidence          `json:"evidence"`
	ToolCalls    []ToolCall          `json:"tool_calls"`
	GeneratedAt  time.Time           `json:"generated_at"`
}

// Incident is the top-level aggregate the state machine advances.
type Incident struct {
	ID           string              `json:"id"`
	Title        string              `json:"title"`
	State        State               `json:"state"`
	Alert        Alert               `json:"alert"`
	Evidence     []Evidence          `json:"evidence"`
	ToolCalls    []ToolCall          `json:"tool_calls"`
	Diagnosis    *Diagnosis          `json:"diagnosis,omitempty"`
	Remediation  *Remediation        `json:"remediation,omitempty"`
	Risk         *RiskAssessment     `json:"risk,omitempty"`
	Verification *VerificationResult `json:"verification,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	History      []Transition        `json:"history"`
}

// Transition is one recorded step of the state machine, for the audit trail.
type Transition struct {
	From   State     `json:"from"`
	To     State     `json:"to"`
	Reason string    `json:"reason"`
	At     time.Time `json:"at"`
}
