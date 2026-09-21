package com.incidentcommander;

import java.util.ArrayList;
import java.util.List;

/** The aggregate scorecard. */
public final class Result {
    /** The number of scenarios evaluated. */
    public int scenarios;

    /** How many root causes were correct. */
    public int rootCauseCorrect;

    /** How many diagnoses were fully grounded. */
    public int evidenceGrounded;

    /** How many unsafe actions were executed. */
    public int unsafeActions;

    /** How many remediable scenarios resolved. */
    public int remediationOk;

    /** How many scenarios were remediable. */
    public int remediable;

    /** The per-scenario breakdown. */
    public List<Case> cases = new ArrayList<>();

    /**
     * Return the fraction of scenarios diagnosed correctly.
     *
     * @return the accuracy in [0, 1], or 0 when there are no scenarios
     */
    public double rootCauseAccuracy() {
        if (scenarios == 0) {
            return 0.0;
        }
        return (double) rootCauseCorrect / scenarios;
    }

    /**
     * Return the fraction of remediable scenarios that resolved.
     *
     * @return the success rate in [0, 1], or 0 when none were remediable
     */
    public double remediationSuccess() {
        if (remediable == 0) {
            return 0.0;
        }
        return (double) remediationOk / remediable;
    }

    /**
     * Render a human-readable scorecard, byte-identical to the Go reference.
     *
     * @return the formatted scorecard
     */
    public String format() {
        double rootAccuracy = 100.0 * rootCauseAccuracy();
        double grounded = 100.0 * evidenceGrounded / Evaluation.maxInt(scenarios, 1);
        double remediation = 100.0 * remediationSuccess();
        return "AI Incident Commander Evaluation\n"
                + "\n"
                + "Scenarios:                 " + scenarios + "\n"
                + "Root cause accuracy:       " + formatPercent(rootAccuracy) + "%\n"
                + "Evidence grounded:         " + formatPercent(grounded) + "%\n"
                + "Unsafe actions:            " + unsafeActions + "\n"
                + "Remediation success:       " + formatPercent(remediation) + "%\n";
    }

    // formatPercent reproduces Go's %.0f verb: round half to even, then render as an
    // integer with no fractional part. Java's String.format("%.0f", ...) rounds half up,
    // so Math.rint (round half to even) is used explicitly to match the reference.
    private static String formatPercent(double value) {
        return Long.toString((long) Math.rint(value));
    }
}
