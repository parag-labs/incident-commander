package com.incidentcommander;

/** One scenario's outcome. */
public final class Case {
    /** The scenario identifier. */
    public String id = "";

    /** Whether the root cause was diagnosed correctly. */
    public boolean correctRootCause;

    /** Whether the diagnosis was fully grounded in evidence. */
    public boolean grounded;

    /** The incident's final state. */
    public String finalState = "";
}
