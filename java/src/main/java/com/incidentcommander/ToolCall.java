package com.incidentcommander;

import java.time.Instant;
import java.util.Map;

/** Records that a tool ran, for the audit trail. */
public final class ToolCall {
    /** The tool name. */
    public String tool = "";

    /** The service it targeted. */
    public String service = "";

    /** The tool's declared safety level. */
    public String safety = "";

    /** Whether it completed successfully. */
    public boolean ok;

    /** Any error message. */
    public String error = "";

    /** How long it took, in milliseconds. */
    public long durationMs;

    /** Optional arguments (unused by the core). */
    public Map<String, Object> args;

    /** When it started (never inspected). */
    public Instant startedAt;
}
