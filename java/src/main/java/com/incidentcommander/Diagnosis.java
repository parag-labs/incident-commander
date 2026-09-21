package com.incidentcommander;

import java.util.ArrayList;
import java.util.List;

/**
 * The validated, structured output of the AI investigator. It is what the orchestrator
 * acts on - never free-form model text.
 */
public final class Diagnosis {
    /** A narrative summary. */
    public String summary = "";

    /** The ranked hypotheses. */
    public List<Hypothesis> hypotheses = new ArrayList<>();

    /** The recommended action. */
    public String recommendedAction = "";

    /** The recommended service. */
    public String recommendedService = "";

    /** The qualitative risk. */
    public String risk = "";

    /** Whether human approval is requested. */
    public boolean needsHumanApproval;

    /**
     * Return the highest-confidence hypothesis, or {@code null} if there are none. Ties
     * keep the first hypothesis seen, matching the reference's strict {@code >}.
     *
     * @return the top hypothesis, or {@code null} when there are none
     */
    public Hypothesis top() {
        Hypothesis best = null;
        double bestConf = -1.0;
        for (Hypothesis h : hypotheses) {
            if (h.confidence > bestConf) {
                best = h;
                bestConf = h.confidence;
            }
        }
        return best;
    }
}
