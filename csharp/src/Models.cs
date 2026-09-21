// Typed domain contracts shared across the incident commander.
//
// The enum-like types are modelled as open string constants exactly like the Go
// reference's `type X string` declarations, so an out-of-band value such as an unknown
// safety level is representable and the policy engine's default branch stays reachable.

using System;
using System.Collections.Generic;

namespace IncidentCommander
{
    /// <summary>Components of the simulated production environment.</summary>
    public static class Services
    {
        /// <summary>The API service.</summary>
        public const string Api = "api";

        /// <summary>The payment service.</summary>
        public const string Payment = "payment";

        /// <summary>The database service.</summary>
        public const string Database = "database";

        /// <summary>The cache service.</summary>
        public const string Cache = "cache";

        /// <summary>The queue service.</summary>
        public const string Queue = "queue";

        /// <summary>The fleet the simulator models, in a stable order.</summary>
        public static readonly IReadOnlyList<string> All =
            new[] { Api, Payment, Database, Cache, Queue };
    }

    /// <summary>How bad an alert looks on arrival.</summary>
    public static class Severity
    {
        /// <summary>Informational.</summary>
        public const string Info = "info";

        /// <summary>A warning.</summary>
        public const string Warning = "warning";

        /// <summary>Critical.</summary>
        public const string Critical = "critical";
    }

    /// <summary>Nodes in the incident state machine.</summary>
    public static class States
    {
        /// <summary>The initial state of every incident.</summary>
        public const string New = "NEW";

        /// <summary>Triage is under way.</summary>
        public const string Triaging = "TRIAGING";

        /// <summary>Investigation is under way.</summary>
        public const string Investigating = "INVESTIGATING";

        /// <summary>A root cause has been identified.</summary>
        public const string RootCauseIdentified = "ROOT_CAUSE_IDENTIFIED";

        /// <summary>A remediation has been proposed.</summary>
        public const string RemediationProposed = "REMEDIATION_PROPOSED";

        /// <summary>The remediation is waiting for human approval.</summary>
        public const string WaitingForApproval = "WAITING_FOR_APPROVAL";

        /// <summary>The remediation is executing.</summary>
        public const string Remediating = "REMEDIATING";

        /// <summary>The remediation is being verified.</summary>
        public const string Verifying = "VERIFYING";

        /// <summary>The incident is resolved (terminal).</summary>
        public const string Resolved = "RESOLVED";

        /// <summary>Investigation failed (terminal).</summary>
        public const string InvestigationFailed = "INVESTIGATION_FAILED";

        /// <summary>Remediation failed (terminal).</summary>
        public const string RemediationFailed = "REMEDIATION_FAILED";

        /// <summary>Verification failed (terminal).</summary>
        public const string VerificationFailed = "VERIFICATION_FAILED";

        /// <summary>The incident was escalated to a human (terminal).</summary>
        public const string Escalated = "ESCALATED";
    }

    /// <summary>How dangerous a tool is.</summary>
    public static class Safety
    {
        /// <summary>Read-only / low risk.</summary>
        public const string Low = "low";

        /// <summary>Medium risk.</summary>
        public const string Medium = "medium";

        /// <summary>High risk.</summary>
        public const string High = "high";
    }

    /// <summary>The qualitative risk the agent assigns to a recommended action.</summary>
    public static class Risk
    {
        /// <summary>Low risk.</summary>
        public const string Low = "low";

        /// <summary>Medium risk.</summary>
        public const string Medium = "medium";

        /// <summary>High risk.</summary>
        public const string High = "high";
    }

    /// <summary>The signal that opens an incident.</summary>
    public sealed class Alert
    {
        /// <summary>The service the alert fired on.</summary>
        public string Service { get; set; } = "";

        /// <summary>The metric that breached.</summary>
        public string Metric { get; set; } = "";

        /// <summary>The observed value.</summary>
        public double Value { get; set; }

        /// <summary>The severity as reported.</summary>
        public string SeverityLevel { get; set; } = "";

        /// <summary>A human-readable message.</summary>
        public string Message { get; set; } = "";

        /// <summary>An optional identifier.</summary>
        public string Id { get; set; } = "";

        /// <summary>When the alert fired (never inspected by the core).</summary>
        public DateTimeOffset FiredAt { get; set; }
    }

    /// <summary>One grounded fact a tool produced.</summary>
    public sealed class Evidence
    {
        /// <summary>The evidence identifier cited by hypotheses.</summary>
        public string Id { get; set; } = "";

        /// <summary>The tool that produced it.</summary>
        public string Tool { get; set; } = "";

        /// <summary>The service it concerns.</summary>
        public string Service { get; set; } = "";

        /// <summary>A human-readable summary.</summary>
        public string Summary { get; set; } = "";

        /// <summary>Optional structured detail (unused by the core).</summary>
        public IDictionary<string, object>? Detail { get; set; }
    }

    /// <summary>A candidate root cause with a confidence and the evidence behind it.</summary>
    public sealed class Hypothesis
    {
        /// <summary>The proposed cause.</summary>
        public string Cause { get; set; } = "";

        /// <summary>The confidence in [0, 1].</summary>
        public double Confidence { get; set; }

        /// <summary>The evidence identifiers cited in support.</summary>
        public IList<string> EvidenceIds { get; set; } = new List<string>();
    }

    /// <summary>Records that a tool ran, for the audit trail.</summary>
    public sealed class ToolCall
    {
        /// <summary>The tool name.</summary>
        public string Tool { get; set; } = "";

        /// <summary>The service it targeted.</summary>
        public string Service { get; set; } = "";

        /// <summary>The tool's declared safety level.</summary>
        public string SafetyLevel { get; set; } = "";

        /// <summary>Whether it completed successfully.</summary>
        public bool Ok { get; set; }

        /// <summary>Any error message.</summary>
        public string Error { get; set; } = "";

        /// <summary>How long it took, in milliseconds.</summary>
        public long DurationMs { get; set; }

        /// <summary>Optional arguments (unused by the core).</summary>
        public IDictionary<string, object>? Args { get; set; }

        /// <summary>When it started (never inspected).</summary>
        public DateTimeOffset StartedAt { get; set; }
    }

    /// <summary>A proposed corrective action targeting one service.</summary>
    public sealed class Remediation
    {
        /// <summary>The action name (matched against tool calls).</summary>
        public string Action { get; set; } = "";

        /// <summary>The service it targets.</summary>
        public string Service { get; set; } = "";

        /// <summary>The declared safety level.</summary>
        public string SafetyLevel { get; set; } = "";

        /// <summary>Why it was proposed.</summary>
        public string Reason { get; set; } = "";
    }

    /// <summary>The deterministic policy engine's verdict on a remediation.</summary>
    public sealed class RiskAssessment
    {
        /// <summary>The safety level assessed.</summary>
        public string SafetyLevel { get; set; } = "";

        /// <summary>Whether the action may run automatically.</summary>
        public bool AutoApprovable { get; set; }

        /// <summary>Whether the action requires human approval.</summary>
        public bool NeedsApproval { get; set; }

        /// <summary>The human-readable justification.</summary>
        public string Reason { get; set; } = "";
    }

    /// <summary>The validated, structured output of the AI investigator.</summary>
    public sealed class Diagnosis
    {
        /// <summary>A narrative summary.</summary>
        public string Summary { get; set; } = "";

        /// <summary>The ranked hypotheses.</summary>
        public IList<Hypothesis> Hypotheses { get; set; } = new List<Hypothesis>();

        /// <summary>The recommended action.</summary>
        public string RecommendedAction { get; set; } = "";

        /// <summary>The recommended service.</summary>
        public string RecommendedService { get; set; } = "";

        /// <summary>The qualitative risk.</summary>
        public string RiskLevel { get; set; } = "";

        /// <summary>Whether human approval is requested.</summary>
        public bool NeedsHumanApproval { get; set; }

        /// <summary>
        /// Return the highest-confidence hypothesis, or <c>null</c> if there are none.
        /// Ties keep the first hypothesis seen, matching the reference's strict &gt;.
        /// </summary>
        public Hypothesis? Top()
        {
            Hypothesis? best = null;
            double bestConf = -1.0;
            foreach (Hypothesis h in Hypotheses)
            {
                if (h.Confidence > bestConf)
                {
                    best = h;
                    bestConf = h.Confidence;
                }
            }

            return best;
        }
    }

    /// <summary>Records whether an executed remediation actually improved things.</summary>
    public sealed class VerificationResult
    {
        /// <summary>Whether the situation improved.</summary>
        public bool Improved { get; set; }

        /// <summary>Error percentage before.</summary>
        public double BeforeErrPct { get; set; }

        /// <summary>Error percentage after.</summary>
        public double AfterErrPct { get; set; }

        /// <summary>p99 latency before, in milliseconds.</summary>
        public double BeforeP99Ms { get; set; }

        /// <summary>p99 latency after, in milliseconds.</summary>
        public double AfterP99Ms { get; set; }

        /// <summary>A human-readable note.</summary>
        public string Note { get; set; } = "";
    }

    /// <summary>One recorded step of the state machine, for the audit trail.</summary>
    public sealed class Transition
    {
        /// <summary>The state moved from (empty for the opening entry).</summary>
        public string From { get; set; } = "";

        /// <summary>The state moved to.</summary>
        public string To { get; set; } = "";

        /// <summary>Why the move happened.</summary>
        public string Reason { get; set; } = "";

        /// <summary>When it happened (never inspected).</summary>
        public DateTimeOffset At { get; set; }
    }

    /// <summary>The evidence-backed narrative produced at the end.</summary>
    public sealed class IncidentReport
    {
        /// <summary>The incident identifier.</summary>
        public string IncidentId { get; set; } = "";

        /// <summary>The incident title.</summary>
        public string Title { get; set; } = "";

        /// <summary>The final state.</summary>
        public string State { get; set; } = "";

        /// <summary>The root cause narrative.</summary>
        public string RootCause { get; set; } = "";

        /// <summary>The diagnosis, if any.</summary>
        public Diagnosis? Diagnosis { get; set; }

        /// <summary>The remediation, if any.</summary>
        public Remediation? Remediation { get; set; }

        /// <summary>The verification result, if any.</summary>
        public VerificationResult? Verification { get; set; }

        /// <summary>The collected evidence.</summary>
        public IList<Evidence> Evidence { get; set; } = new List<Evidence>();

        /// <summary>The recorded tool calls.</summary>
        public IList<ToolCall> ToolCalls { get; set; } = new List<ToolCall>();

        /// <summary>When the report was generated.</summary>
        public DateTimeOffset GeneratedAt { get; set; }
    }

    /// <summary>The top-level aggregate the state machine advances.</summary>
    public sealed class Incident
    {
        /// <summary>The incident identifier.</summary>
        public string Id { get; set; } = "";

        /// <summary>The incident title.</summary>
        public string Title { get; set; } = "";

        /// <summary>The current state.</summary>
        public string State { get; set; } = "";

        /// <summary>The alert that opened it.</summary>
        public Alert Alert { get; set; } = new Alert();

        /// <summary>The collected evidence.</summary>
        public IList<Evidence> Evidence { get; set; } = new List<Evidence>();

        /// <summary>The recorded tool calls.</summary>
        public IList<ToolCall> ToolCalls { get; set; } = new List<ToolCall>();

        /// <summary>The diagnosis, if any.</summary>
        public Diagnosis? Diagnosis { get; set; }

        /// <summary>The remediation, if any.</summary>
        public Remediation? Remediation { get; set; }

        /// <summary>The policy verdict, if any.</summary>
        public RiskAssessment? Risk { get; set; }

        /// <summary>The verification result, if any.</summary>
        public VerificationResult? Verification { get; set; }

        /// <summary>When it was created.</summary>
        public DateTimeOffset CreatedAt { get; set; }

        /// <summary>When it was last updated.</summary>
        public DateTimeOffset UpdatedAt { get; set; }

        /// <summary>The full transition history.</summary>
        public IList<Transition> History { get; set; } = new List<Transition>();
    }
}
