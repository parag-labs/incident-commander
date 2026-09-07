package agent

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/parag-labs/incident-commander/internal/simulator"
	"github.com/parag-labs/incident-commander/internal/tools"
	"github.com/parag-labs/incident-commander/pkg/models"
)

// job is one unit of investigation work with a stable index so results can be
// re-ordered deterministically after the concurrent phase.
type job struct {
	idx     int
	tool    string
	service models.Service
}

// jobResult carries a completed job's evidence and audit record back to the aggregator.
type jobResult struct {
	idx      int
	evidence []models.Evidence
	call     models.ToolCall
}

// Collector gathers evidence from read-only tools concurrently with a bounded worker
// pool. This is the Go-concurrency core of an investigation: work is fanned across
// workers, context cancellation is respected, and results are aggregated in a stable
// order so the evidence IDs are deterministic despite the parallelism.
type Collector struct {
	registry *tools.Registry
	workers  int
}

// NewCollector builds a collector with the given pool size (minimum 1).
func NewCollector(registry *tools.Registry, workers int) *Collector {
	if workers < 1 {
		workers = 1
	}
	return &Collector{registry: registry, workers: workers}
}

// plan builds the stable list of investigation jobs for an incident.
func (c *Collector) plan(alertService models.Service) []job {
	var jobs []job
	add := func(tool string, svc models.Service) {
		jobs = append(jobs, job{idx: len(jobs), tool: tool, service: svc})
	}
	// Per-service signals, collected for the whole fleet in parallel.
	for _, svc := range models.AllServices {
		add("get_service_health", svc)
		add("query_metrics", svc)
		add("search_logs", svc)
	}
	// Fleet-wide / targeted one-offs.
	add("get_recent_deployments", alertService)
	add("get_dependency_graph", alertService)
	add("get_database_health", models.ServiceDatabase)
	add("get_cache_health", models.ServiceCache)
	return jobs
}

// Collect runs the plan concurrently and returns aggregated evidence and audit records.
// Evidence IDs are assigned after the concurrent phase in job order, so the output is
// identical every run even though the tools execute in parallel.
func (c *Collector) Collect(ctx context.Context, env *simulator.Env, alertService models.Service, now func() time.Time) ([]models.Evidence, []models.ToolCall) {
	jobs := c.plan(alertService)
	jobCh := make(chan job)
	resCh := make(chan jobResult, len(jobs))

	var wg sync.WaitGroup
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				// A no-op ID function during collection; real IDs are assigned in order
				// after aggregation so parallelism can't perturb them.
				ev, call, _ := c.registry.Collect(ctx, env, j.tool, j.service, func() string { return "" }, now)
				resCh <- jobResult{idx: j.idx, evidence: ev, call: call}
			}
		}()
	}

	go func() {
		defer close(jobCh)
		for _, j := range jobs {
			select {
			case <-ctx.Done():
				return
			case jobCh <- j:
			}
		}
	}()

	go func() { wg.Wait(); close(resCh) }()

	results := make([]jobResult, 0, len(jobs))
	for r := range resCh {
		results = append(results, r)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].idx < results[j].idx })

	var evidence []models.Evidence
	var calls []models.ToolCall
	n := 0
	for _, r := range results {
		calls = append(calls, r.call)
		for _, e := range r.evidence {
			n++
			e.ID = fmt.Sprintf("e%d", n)
			evidence = append(evidence, e)
		}
	}
	return evidence, calls
}
