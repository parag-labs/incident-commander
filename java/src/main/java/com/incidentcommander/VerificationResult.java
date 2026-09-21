package com.incidentcommander;

/** Records whether an executed remediation actually improved things. */
public final class VerificationResult {
    /** Whether the situation improved. */
    public boolean improved;

    /** Error percentage before. */
    public double beforeErrPct;

    /** Error percentage after. */
    public double afterErrPct;

    /** p99 latency before, in milliseconds. */
    public double beforeP99Ms;

    /** p99 latency after, in milliseconds. */
    public double afterP99Ms;

    /** A human-readable note. */
    public String note = "";
}
