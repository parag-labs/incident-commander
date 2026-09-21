package com.incidentcommander;

/** The deterministic policy engine's verdict on a remediation. */
public final class RiskAssessment {
    /** The safety level assessed. */
    public String safety = "";

    /** Whether the action may run automatically. */
    public boolean autoApprovable;

    /** Whether the action requires human approval. */
    public boolean needsApproval;

    /** The human-readable justification. */
    public String reason = "";
}
