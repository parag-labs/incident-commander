package com.incidentcommander;

import java.time.Instant;

/** One recorded step of the state machine, for the audit trail. */
public final class Transition {
    /** The state moved from (empty for the opening entry). */
    public String from = "";

    /** The state moved to. */
    public String to = "";

    /** Why the move happened. */
    public String reason = "";

    /** When it happened (never inspected). */
    public Instant at;
}
