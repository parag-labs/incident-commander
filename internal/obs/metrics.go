// Package obs is the observability surface: a tiny, dependency-free counter/gauge
// registry that exposes Prometheus text-format metrics, plus a helper for structured
// logging. It deliberately avoids pulling in a metrics framework - the spec asks for
// Prometheus-compatible output, and the standard library plus a small registry is
// enough for that without the dependency.
package obs

import (
	"fmt"
	"sort"
	"sync"
)

// Metrics is a concurrency-safe registry of named counters and gauges.
type Metrics struct {
	mu       sync.Mutex
	counters map[string]float64
	gauges   map[string]float64
}

// NewMetrics builds an empty registry.
func NewMetrics() *Metrics {
	return &Metrics{counters: map[string]float64{}, gauges: map[string]float64{}}
}

// Inc adds delta to a counter.
func (m *Metrics) Inc(name string, delta float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += delta
}

// Set records a gauge value.
func (m *Metrics) Set(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

// Counter returns the current value of a counter.
func (m *Metrics) Counter(name string) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[name]
}

// Expose renders the registry in Prometheus text exposition format.
func (m *Metrics) Expose() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var b []byte
	write := func(kind string, vals map[string]float64) {
		names := make([]string, 0, len(vals))
		for n := range vals {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			b = append(b, fmt.Sprintf("# TYPE %s %s\n%s %g\n", n, kind, n, vals[n])...)
		}
	}
	write("counter", m.counters)
	write("gauge", m.gauges)
	return string(b)
}
