package com.incidentcommander;

import java.util.ArrayList;
import java.util.List;

/** A candidate root cause with a confidence and the evidence behind it. */
public final class Hypothesis {
    /** The proposed cause. */
    public String cause = "";

    /** The confidence in [0, 1]. */
    public double confidence;

    /** The evidence identifiers cited in support. */
    public List<String> evidenceIds = new ArrayList<>();
}
