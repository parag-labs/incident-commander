package com.incidentcommander;

import java.time.Instant;

/** The signal that opens an incident. */
public final class Alert {
    /** An optional identifier. */
    public String id = "";

    /** The service the alert fired on. */
    public String service = "";

    /** The metric that breached. */
    public String metric = "";

    /** The observed value. */
    public double value;

    /** The severity as reported. */
    public String severity = "";

    /** A human-readable message. */
    public String message = "";

    /** When the alert fired (never inspected by the core). */
    public Instant firedAt;
}
