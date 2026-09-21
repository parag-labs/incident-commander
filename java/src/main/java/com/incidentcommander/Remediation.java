package com.incidentcommander;

/** A proposed corrective action targeting one service. */
public final class Remediation {
    /** The action name (matched against tool calls). */
    public String action = "";

    /** The service it targets. */
    public String service = "";

    /** The declared safety level. */
    public String safety = "";

    /** Why it was proposed. */
    public String reason = "";
}
