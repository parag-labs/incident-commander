package com.incidentcommander;

/**
 * The deterministic risk gate between the AI's recommendation and any action.
 *
 * <p>The agent can recommend anything; nothing runs until this engine has classified it
 * and decided whether it may run automatically. This is the hard boundary that keeps
 * free-form model output from ever executing an infrastructure command on its own.
 */
public final class Policy {
    private Policy() {
    }

    /**
     * Controls how permissive the gate is. {@link #autoApproveMedium} lets MEDIUM-safety
     * remediations run without a human. LOW always may; HIGH never may, regardless of this
     * flag.
     */
    public static final class Config {
        /** Whether MEDIUM-safety remediations may run without a human. */
        public boolean autoApproveMedium;

        /** Create a config with auto-approval of MEDIUM disabled. */
        public Config() {
        }

        /**
         * Create a config with the given MEDIUM auto-approval setting.
         *
         * @param autoApproveMedium whether MEDIUM-safety remediations may run automatically
         */
        public Config(boolean autoApproveMedium) {
            this.autoApproveMedium = autoApproveMedium;
        }
    }

    /**
     * Classify a remediation and decide whether it may run automatically. The safety level
     * is the tool's declared level (looked up by the caller), not anything the model
     * asserts. {@code rem} is part of the audit contract but does not change the verdict,
     * matching the reference.
     *
     * @param rem the proposed remediation (kept for signature parity; unused by the verdict)
     * @param safety the tool's declared safety level
     * @param cfg the policy configuration
     * @return the risk assessment verdict
     */
    public static RiskAssessment assess(Remediation rem, String safety, Config cfg) {
        // rem is intentionally unused by the verdict; kept for signature parity.
        RiskAssessment ra = new RiskAssessment();
        ra.safety = safety;
        if (Safety.LOW.equals(safety)) {
            ra.autoApprovable = true;
            ra.reason = "read-only / low-risk action; safe to run automatically";
        } else if (Safety.MEDIUM.equals(safety)) {
            if (cfg.autoApproveMedium) {
                ra.autoApprovable = true;
                ra.reason = "medium-risk action; auto-approval is enabled";
            } else {
                ra.needsApproval = true;
                ra.reason = "medium-risk action; requires human approval (auto-approval disabled)";
            }
        } else if (Safety.HIGH.equals(safety)) {
            ra.needsApproval = true;
            ra.reason = "high-risk action; never runs without explicit human approval";
        } else {
            ra.needsApproval = true;
            ra.reason = "unknown safety level \"" + safety + "\"; defaulting to human approval";
        }
        return ra;
    }
}
