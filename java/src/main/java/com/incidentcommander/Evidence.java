package com.incidentcommander;

import java.util.Map;

/**
 * One grounded fact a tool produced. Every hypothesis the AI makes must cite Evidence IDs
 * that actually exist - that is how hallucination is caught.
 */
public final class Evidence {
    /** The evidence identifier cited by hypotheses. */
    public String id = "";

    /** The tool that produced it. */
    public String tool = "";

    /** The service it concerns. */
    public String service = "";

    /** A human-readable summary. */
    public String summary = "";

    /** Optional structured detail (unused by the core). */
    public Map<String, Object> detail;
}
