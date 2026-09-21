package com.incidentcommander;

import java.util.List;

/**
 * Components of the simulated production environment.
 *
 * <p>Modelled as open string constants exactly like the Go reference's {@code type Service
 * string}, so serialization stays readable and unknown values remain representable.
 */
public final class Services {
    private Services() {
    }

    /** The API service. */
    public static final String API = "api";

    /** The payment service. */
    public static final String PAYMENT = "payment";

    /** The database service. */
    public static final String DATABASE = "database";

    /** The cache service. */
    public static final String CACHE = "cache";

    /** The queue service. */
    public static final String QUEUE = "queue";

    /** The fleet the simulator models, in a stable order. */
    public static final List<String> ALL = List.of(API, PAYMENT, DATABASE, CACHE, QUEUE);
}
