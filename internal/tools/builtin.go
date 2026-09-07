package tools

import (
	"fmt"

	"github.com/parag-labs/incident-commander/internal/simulator"
	"github.com/parag-labs/incident-commander/pkg/models"
)

// defaultInvestigators returns the read-only evidence-collecting tools. Each one turns
// a slice of the simulated environment into typed Evidence with a stable ID, so a
// hypothesis can cite exactly which fact it rests on.
func defaultInvestigators() []Investigator {
	return []Investigator{
		{
			Name: "get_service_health", Safety: models.SafetyLow,
			Desc: "overall health snapshot for a service",
			fn: func(env *simulator.Env, svc models.Service, id func() string) []models.Evidence {
				m := env.Snapshot(svc)
				return []models.Evidence{{
					ID: id(), Tool: "get_service_health", Service: svc,
					Summary: fmt.Sprintf("%s healthy=%v error_rate=%.1f%% p99=%.0fms", svc, m.Healthy, m.ErrorRatePct, m.LatencyP99MS),
					Detail:  metricDetail(m),
				}}
			},
		},
		{
			Name: "query_metrics", Safety: models.SafetyLow,
			Desc: "latency, error rate, CPU and memory for a service",
			fn: func(env *simulator.Env, svc models.Service, id func() string) []models.Evidence {
				m := env.Snapshot(svc)
				return []models.Evidence{{
					ID: id(), Tool: "query_metrics", Service: svc,
					Summary: fmt.Sprintf("%s p99=%.0fms err=%.1f%% cpu=%.0f%% mem=%.0f%%", svc, m.LatencyP99MS, m.ErrorRatePct, m.CPUPct, m.MemoryPct),
					Detail:  metricDetail(m),
				}}
			},
		},
		{
			Name: "search_logs", Safety: models.SafetyLow,
			Desc: "recent error log signal for a service",
			fn: func(env *simulator.Env, svc models.Service, id func() string) []models.Evidence {
				m := env.Snapshot(svc)
				line := "no elevated error signal"
				if m.ErrorRatePct >= 5 {
					line = fmt.Sprintf("elevated 5xx rate ~%.0f%% in the last 5m", m.ErrorRatePct)
				}
				return []models.Evidence{{
					ID: id(), Tool: "search_logs", Service: svc,
					Summary: fmt.Sprintf("%s logs: %s", svc, line),
					Detail:  map[string]any{"error_rate_pct": m.ErrorRatePct},
				}}
			},
		},
		{
			Name: "get_recent_deployments", Safety: models.SafetyLow,
			Desc: "recent deployments across the fleet",
			fn: func(env *simulator.Env, svc models.Service, id func() string) []models.Evidence {
				deploys := env.Deployments()
				summary := "no recent deployments"
				detail := map[string]any{}
				if len(deploys) > 0 {
					d := deploys[0]
					summary = fmt.Sprintf("most recent deploy: %s %s %dm ago", d.Service, d.Version, d.MinsAgo)
					detail = map[string]any{"service": d.Service, "version": d.Version, "minutes_ago": d.MinsAgo}
				}
				return []models.Evidence{{
					ID: id(), Tool: "get_recent_deployments", Service: svc,
					Summary: summary, Detail: detail,
				}}
			},
		},
		{
			Name: "get_dependency_graph", Safety: models.SafetyLow,
			Desc: "direct dependencies of a service",
			fn: func(env *simulator.Env, svc models.Service, id func() string) []models.Evidence {
				deps := env.Dependencies(svc)
				return []models.Evidence{{
					ID: id(), Tool: "get_dependency_graph", Service: svc,
					Summary: fmt.Sprintf("%s depends on %v", svc, deps),
					Detail:  map[string]any{"dependencies": deps},
				}}
			},
		},
		{
			Name: "get_database_health", Safety: models.SafetyLow,
			Desc: "database connection pool health",
			fn: func(env *simulator.Env, svc models.Service, id func() string) []models.Evidence {
				m := env.Snapshot(models.ServiceDatabase)
				return []models.Evidence{{
					ID: id(), Tool: "get_database_health", Service: models.ServiceDatabase,
					Summary: fmt.Sprintf("database connections=%d p99=%.0fms", m.DBConns, m.LatencyP99MS),
					Detail:  map[string]any{"db_connections": m.DBConns, "latency_p99_ms": m.LatencyP99MS},
				}}
			},
		},
		{
			Name: "get_cache_health", Safety: models.SafetyLow,
			Desc: "cache hit rate and latency",
			fn: func(env *simulator.Env, svc models.Service, id func() string) []models.Evidence {
				m := env.Snapshot(models.ServiceCache)
				return []models.Evidence{{
					ID: id(), Tool: "get_cache_health", Service: models.ServiceCache,
					Summary: fmt.Sprintf("cache hit_rate=%.0f%% p99=%.0fms", m.CacheHitPct, m.LatencyP99MS),
					Detail:  map[string]any{"cache_hit_rate_pct": m.CacheHitPct, "latency_p99_ms": m.LatencyP99MS},
				}}
			},
		},
	}
}

// defaultRemediators returns the state-changing tools with their safety levels. LOW is
// read-only (none here); MEDIUM is disruptive-but-recoverable; HIGH is dangerous and
// never runs automatically.
func defaultRemediators() []Remediator {
	apply := func(action string) func(*simulator.Env, models.Service) bool {
		return func(env *simulator.Env, svc models.Service) bool { return env.Apply(action, svc) }
	}
	return []Remediator{
		{Name: "clear_cache", Safety: models.SafetyMedium, Desc: "flush the cache", fn: apply("clear_cache")},
		{Name: "restart_service", Safety: models.SafetyMedium, Desc: "restart a service", fn: apply("restart_service")},
		{Name: "scale_service", Safety: models.SafetyMedium, Desc: "add replicas to a service", fn: apply("scale_service")},
		{Name: "reduce_db_connections", Safety: models.SafetyMedium, Desc: "shed and cap database connections", fn: apply("reduce_db_connections")},
		{Name: "rollback_deployment", Safety: models.SafetyHigh, Desc: "roll back a production deployment", fn: apply("rollback_deployment")},
	}
}

func metricDetail(m simulator.Metrics) map[string]any {
	return map[string]any{
		"latency_p99_ms":     m.LatencyP99MS,
		"error_rate_pct":     m.ErrorRatePct,
		"cpu_pct":            m.CPUPct,
		"memory_pct":         m.MemoryPct,
		"queue_depth":        m.QueueDepth,
		"db_connections":     m.DBConns,
		"cache_hit_rate_pct": m.CacheHitPct,
		"healthy":            m.Healthy,
	}
}
