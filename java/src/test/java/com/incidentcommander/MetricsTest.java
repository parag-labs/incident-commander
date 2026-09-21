package com.incidentcommander;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;

class MetricsTest {
    @Test
    void counterAccumulates() {
        Metrics m = Metrics.newMetrics();
        m.inc("incident_count", 1);
        m.inc("incident_count", 2);
        assertEquals(3.0, m.counter("incident_count"));
    }

    @Test
    void counterMissingIsZero() {
        assertEquals(0.0, Metrics.newMetrics().counter("nope"));
    }

    @Test
    void exposeContainsCounterGaugeAndTypeLines() {
        Metrics m = Metrics.newMetrics();
        m.inc("incident_count", 1);
        m.inc("incident_count", 2);
        m.set("db_connections", 42.0);
        String out = m.expose();
        assertTrue(out.contains("incident_count 3"));
        assertTrue(out.contains("db_connections 42"));
        assertTrue(out.contains("# TYPE incident_count counter"));
        assertTrue(out.contains("# TYPE db_connections gauge"));
    }

    @Test
    void exposeIsSortedByName() {
        Metrics m = Metrics.newMetrics();
        m.inc("b_metric", 1);
        m.inc("a_metric", 1);
        String out = m.expose();
        assertTrue(out.indexOf("a_metric") < out.indexOf("b_metric"));
    }

    @Test
    void exposeExactLayout() {
        Metrics m = Metrics.newMetrics();
        m.inc("z_counter", 3);
        m.inc("a_counter", 1);
        m.set("b_gauge", 1.5);
        m.set("a_gauge", 0.2);
        String expected =
                "# TYPE a_counter counter\n"
                + "a_counter 1\n"
                + "# TYPE z_counter counter\n"
                + "z_counter 3\n"
                + "# TYPE a_gauge gauge\n"
                + "a_gauge 0.2\n"
                + "# TYPE b_gauge gauge\n"
                + "b_gauge 1.5\n";
        assertEquals(expected, m.expose());
    }

    @Test
    void exposeEmptyIsBlank() {
        assertEquals("", Metrics.newMetrics().expose());
    }

    @Test
    void setOverwritesGauge() {
        Metrics m = Metrics.newMetrics();
        m.set("queue_depth", 3.0);
        m.set("queue_depth", 1.0);
        assertTrue(m.expose().contains("queue_depth 1\n"));
    }

    @Test
    void formatGIntegers() {
        assertEquals("0", Metrics.formatG(0.0));
        assertEquals("1", Metrics.formatG(1.0));
        assertEquals("3", Metrics.formatG(3.0));
        assertEquals("42", Metrics.formatG(42.0));
        assertEquals("1000", Metrics.formatG(1000.0));
    }

    @Test
    void formatGSimpleDecimals() {
        assertEquals("0.2", Metrics.formatG(0.2));
        assertEquals("1.5", Metrics.formatG(1.5));
        assertEquals("0.0001", Metrics.formatG(0.0001));
        assertEquals("0.25", Metrics.formatG(0.25));
    }

    @Test
    void formatGNegative() {
        assertEquals("-1", Metrics.formatG(-1.0));
        assertEquals("-0.5", Metrics.formatG(-0.5));
    }
}
