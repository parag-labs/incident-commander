package com.incidentcommander;

/**
 * Classifies how dangerous a tool is. Decides whether the deterministic policy engine will
 * let the AI's recommendation run automatically. Open string constants, like the Go
 * reference.
 */
public final class Safety {
    private Safety() {
    }

    /** Read-only; always safe to run automatically. */
    public static final String LOW = "low";

    /** Potentially disruptive; runs only if auto-approval is enabled. */
    public static final String MEDIUM = "medium";

    /** Dangerous; never runs without a human. */
    public static final String HIGH = "high";
}
