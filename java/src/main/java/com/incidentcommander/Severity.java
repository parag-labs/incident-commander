package com.incidentcommander;

/** How bad an alert looks on arrival. Open string constants, like the Go reference. */
public final class Severity {
    private Severity() {
    }

    /** Informational. */
    public static final String INFO = "info";

    /** A warning. */
    public static final String WARNING = "warning";

    /** Critical. */
    public static final String CRITICAL = "critical";
}
