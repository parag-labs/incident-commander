package obs

import (
	"strings"
	"testing"
)

func TestCounterAndGauge(t *testing.T) {
	m := NewMetrics()
	m.Inc("incident_count", 1)
	m.Inc("incident_count", 2)
	if got := m.Counter("incident_count"); got != 3 {
		t.Fatalf("want 3, got %g", got)
	}
	m.Set("db_connections", 42)

	out := m.Expose()
	if !strings.Contains(out, "incident_count 3") {
		t.Fatalf("expose missing counter: %q", out)
	}
	if !strings.Contains(out, "db_connections 42") {
		t.Fatalf("expose missing gauge: %q", out)
	}
	if !strings.Contains(out, "# TYPE incident_count counter") {
		t.Fatalf("expose missing TYPE line: %q", out)
	}
}

func TestExposeIsStable(t *testing.T) {
	m := NewMetrics()
	m.Inc("b_metric", 1)
	m.Inc("a_metric", 1)
	// Names are sorted, so output is stable regardless of insertion order.
	out := m.Expose()
	if strings.Index(out, "a_metric") > strings.Index(out, "b_metric") {
		t.Fatalf("metrics should be sorted by name: %q", out)
	}
}
