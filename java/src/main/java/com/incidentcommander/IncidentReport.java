package com.incidentcommander;

import java.time.Instant;
import java.util.ArrayList;
import java.util.List;

/** The evidence-backed narrative produced at the end. */
public final class IncidentReport {
    /** The incident identifier. */
    public String incidentId = "";

    /** The incident title. */
    public String title = "";

    /** The final state. */
    public String state = "";

    /** The root cause narrative. */
    public String rootCause = "";

    /** The diagnosis, if any. */
    public Diagnosis diagnosis;

    /** The remediation, if any. */
    public Remediation remediation;

    /** The verification result, if any. */
    public VerificationResult verification;

    /** The collected evidence. */
    public List<Evidence> evidence = new ArrayList<>();

    /** The recorded tool calls. */
    public List<ToolCall> toolCalls = new ArrayList<>();

    /** When the report was generated. */
    public Instant generatedAt;
}
