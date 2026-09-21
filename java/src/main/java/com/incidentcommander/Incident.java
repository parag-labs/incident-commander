package com.incidentcommander;

import java.time.Instant;
import java.util.ArrayList;
import java.util.List;

/** The top-level aggregate the state machine advances. */
public final class Incident {
    /** The incident identifier. */
    public String id = "";

    /** The incident title. */
    public String title = "";

    /** The current state. */
    public String state = "";

    /** The alert that opened it. */
    public Alert alert = new Alert();

    /** The collected evidence. */
    public List<Evidence> evidence = new ArrayList<>();

    /** The recorded tool calls. */
    public List<ToolCall> toolCalls = new ArrayList<>();

    /** The diagnosis, if any. */
    public Diagnosis diagnosis;

    /** The remediation, if any. */
    public Remediation remediation;

    /** The policy verdict, if any. */
    public RiskAssessment risk;

    /** The verification result, if any. */
    public VerificationResult verification;

    /** When it was created. */
    public Instant createdAt;

    /** When it was last updated. */
    public Instant updatedAt;

    /** The full transition history. */
    public List<Transition> history = new ArrayList<>();
}
