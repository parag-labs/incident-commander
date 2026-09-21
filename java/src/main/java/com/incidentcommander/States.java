package com.incidentcommander;

/**
 * Nodes in the incident state machine.
 *
 * <p>The machine is deterministic and the LLM never sets these directly - only the
 * orchestrator advances an incident through validated transitions.
 */
public final class States {
    private States() {
    }

    /** The initial state of every incident. */
    public static final String NEW = "NEW";

    /** Triage is under way. */
    public static final String TRIAGING = "TRIAGING";

    /** Investigation is under way. */
    public static final String INVESTIGATING = "INVESTIGATING";

    /** A root cause has been identified. */
    public static final String ROOT_CAUSE_IDENTIFIED = "ROOT_CAUSE_IDENTIFIED";

    /** A remediation has been proposed. */
    public static final String REMEDIATION_PROPOSED = "REMEDIATION_PROPOSED";

    /** The remediation is waiting for human approval. */
    public static final String WAITING_FOR_APPROVAL = "WAITING_FOR_APPROVAL";

    /** The remediation is executing. */
    public static final String REMEDIATING = "REMEDIATING";

    /** The remediation is being verified. */
    public static final String VERIFYING = "VERIFYING";

    /** The incident is resolved (terminal). */
    public static final String RESOLVED = "RESOLVED";

    /** Investigation failed (terminal). */
    public static final String INVESTIGATION_FAILED = "INVESTIGATION_FAILED";

    /** Remediation failed (terminal). */
    public static final String REMEDIATION_FAILED = "REMEDIATION_FAILED";

    /** Verification failed (terminal). */
    public static final String VERIFICATION_FAILED = "VERIFICATION_FAILED";

    /** The incident was escalated to a human (terminal). */
    public static final String ESCALATED = "ESCALATED";
}
