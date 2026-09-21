package com.incidentcommander;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;

import java.util.List;
import org.junit.jupiter.api.Test;

class ModelsTest {
    @Test
    void allServicesStableOrder() {
        assertEquals(List.of("api", "payment", "database", "cache", "queue"), Services.ALL);
    }

    @Test
    void stateConstantsAreReadableStrings() {
        assertEquals("NEW", States.NEW);
        assertEquals("WAITING_FOR_APPROVAL", States.WAITING_FOR_APPROVAL);
        assertEquals("low", Safety.LOW);
        assertEquals("critical", Severity.CRITICAL);
    }

    private static Hypothesis hyp(String cause, double confidence) {
        Hypothesis h = new Hypothesis();
        h.cause = cause;
        h.confidence = confidence;
        return h;
    }

    @Test
    void diagnosisTopReturnsHighestConfidence() {
        Diagnosis d = new Diagnosis();
        d.hypotheses = List.of(hyp("a", 0.3), hyp("b", 0.9), hyp("c", 0.5));
        assertEquals("b", d.top().cause);
    }

    @Test
    void diagnosisTopTiesKeepFirst() {
        Diagnosis d = new Diagnosis();
        d.hypotheses = List.of(hyp("first", 0.8), hyp("second", 0.8));
        assertEquals("first", d.top().cause);
    }

    @Test
    void diagnosisTopEmptyIsNull() {
        assertNull(new Diagnosis().top());
    }

    @Test
    void diagnosisTopHandlesZeroConfidence() {
        Diagnosis d = new Diagnosis();
        d.hypotheses = List.of(hyp("only", 0.0));
        assertEquals("only", d.top().cause);
    }
}
