package simulator

import "github.com/parag-labs/incident-commander/pkg/models"

// Scenario is a deterministic, reproducible incident: which services are degraded and
// how, what alert fires, and the single (action, service) pair that actually fixes it.
// The FixAction/FixService pair is the ground truth the evaluation harness scores
// against - the agent doesn't see it; it has to reach it from the evidence.
type Scenario struct {
	ID          string
	Title       string
	Alert       models.Alert
	Faults      map[models.Service]func(*Metrics)
	Deployments []Deployment
	FixAction   string
	FixService  models.Service
	// RootCause is the human-readable ground-truth cause, for reporting and scoring.
	RootCause string
}

func alert(svc models.Service, metric string, value float64) models.Alert {
	return models.Alert{
		Service:  svc,
		Metric:   metric,
		Value:    value,
		Severity: models.SeverityCritical,
		Message:  string(svc) + " " + metric + " breached threshold",
	}
}

// Scenarios is the fixed catalogue of incidents used by tests, the demo, and the
// evaluation harness. Order is stable.
var Scenarios = []Scenario{
	{
		ID:        "database_overload",
		Title:     "Database connection pool exhausted",
		Alert:     alert(models.ServiceDatabase, "db_connections", 100),
		FixAction: "reduce_db_connections", FixService: models.ServiceDatabase,
		RootCause: "database connection pool exhausted, saturating query latency",
		Faults: map[models.Service]func(*Metrics){
			models.ServiceDatabase: func(m *Metrics) {
				m.DBConns = 100
				m.LatencyP99MS = 2500
				m.ErrorRatePct = 18
				m.Healthy = false
			},
		},
	},
	{
		ID:        "cache_failure",
		Title:     "Cache hit rate collapsed",
		Alert:     alert(models.ServiceCache, "cache_hit_rate_pct", 8),
		FixAction: "clear_cache", FixService: models.ServiceCache,
		RootCause: "cache holding poisoned/stale entries, hit rate collapsed",
		Faults: map[models.Service]func(*Metrics){
			models.ServiceCache: func(m *Metrics) {
				m.CacheHitPct = 8
				m.LatencyP99MS = 400
				m.Healthy = false
			},
		},
	},
	{
		ID:        "deployment_regression",
		Title:     "Error spike after API deploy",
		Alert:     alert(models.ServiceAPI, "error_rate_pct", 22),
		FixAction: "rollback_deployment", FixService: models.ServiceAPI,
		RootCause:   "recent api deployment introduced a regression, error rate spiked",
		Deployments: []Deployment{{Service: models.ServiceAPI, Version: "v2.4.1", MinsAgo: 6}},
		Faults: map[models.Service]func(*Metrics){
			models.ServiceAPI: func(m *Metrics) {
				m.ErrorRatePct = 22
				m.LatencyP99MS = 600
				m.Healthy = false
			},
		},
	},
	{
		ID:        "queue_backlog",
		Title:     "Queue depth runaway",
		Alert:     alert(models.ServiceQueue, "queue_depth", 800),
		FixAction: "scale_service", FixService: models.ServiceQueue,
		RootCause: "queue consumers under-provisioned, depth backed up",
		Faults: map[models.Service]func(*Metrics){
			models.ServiceQueue: func(m *Metrics) {
				m.QueueDepth = 800
				m.LatencyP99MS = 900
				m.Healthy = false
			},
		},
	},
	{
		ID:        "memory_leak",
		Title:     "Payment service memory leak",
		Alert:     alert(models.ServicePayment, "memory_pct", 96),
		FixAction: "restart_service", FixService: models.ServicePayment,
		RootCause: "payment service leaking memory, approaching OOM",
		Faults: map[models.Service]func(*Metrics){
			models.ServicePayment: func(m *Metrics) {
				m.MemoryPct = 96
				m.LatencyP99MS = 700
				m.ErrorRatePct = 5
				m.Healthy = false
			},
		},
	},
	{
		ID:        "dependency_timeout",
		Title:     "API slow due to payment dependency",
		Alert:     alert(models.ServiceAPI, "latency_p99_ms", 1800),
		FixAction: "restart_service", FixService: models.ServicePayment,
		RootCause: "payment dependency timing out, back-pressuring api latency",
		Faults: map[models.Service]func(*Metrics){
			models.ServicePayment: func(m *Metrics) {
				m.LatencyP99MS = 3000
				m.ErrorRatePct = 12
				m.Healthy = false
			},
			models.ServiceAPI: func(m *Metrics) {
				m.LatencyP99MS = 1800
				m.Healthy = false
			},
		},
	},
	{
		ID:        "high_cpu",
		Title:     "API CPU saturation",
		Alert:     alert(models.ServiceAPI, "cpu_pct", 97),
		FixAction: "scale_service", FixService: models.ServiceAPI,
		RootCause: "api CPU-bound under load, needs horizontal scale-out",
		Faults: map[models.Service]func(*Metrics){
			models.ServiceAPI: func(m *Metrics) {
				m.CPUPct = 97
				m.LatencyP99MS = 1100
				m.Healthy = false
			},
		},
	},
	{
		ID:        "network_latency",
		Title:     "Cache network latency (hit rate healthy)",
		Alert:     alert(models.ServiceCache, "latency_p99_ms", 350),
		FixAction: "restart_service", FixService: models.ServiceCache,
		RootCause: "cache node network path degraded; hit rate normal so it isn't eviction",
		Faults: map[models.Service]func(*Metrics){
			models.ServiceCache: func(m *Metrics) {
				m.LatencyP99MS = 350
				m.CacheHitPct = 93 // still healthy - distinguishes from cache_failure
				m.Healthy = false
			},
		},
	},
	{
		ID:        "auth_failure",
		Title:     "Payment auth errors after deploy",
		Alert:     alert(models.ServicePayment, "error_rate_pct", 27),
		FixAction: "rollback_deployment", FixService: models.ServicePayment,
		RootCause:   "payment deploy broke an auth credential path, errors spiked",
		Deployments: []Deployment{{Service: models.ServicePayment, Version: "v1.9.0", MinsAgo: 4}},
		Faults: map[models.Service]func(*Metrics){
			models.ServicePayment: func(m *Metrics) {
				m.ErrorRatePct = 27
				m.Healthy = false
			},
		},
	},
	{
		ID:        "cascading_failure",
		Title:     "Cascade from database into API",
		Alert:     alert(models.ServiceAPI, "error_rate_pct", 20),
		FixAction: "reduce_db_connections", FixService: models.ServiceDatabase,
		RootCause: "database saturation cascading into api errors; database is the root",
		Faults: map[models.Service]func(*Metrics){
			models.ServiceDatabase: func(m *Metrics) {
				m.DBConns = 100
				m.LatencyP99MS = 2600
				m.ErrorRatePct = 21
				m.Healthy = false
			},
			models.ServiceAPI: func(m *Metrics) {
				m.ErrorRatePct = 20
				m.LatencyP99MS = 1400
				m.Healthy = false
			},
		},
	},
}

// ScenarioByID returns a scenario by its ID.
func ScenarioByID(id string) (Scenario, bool) {
	for _, s := range Scenarios {
		if s.ID == id {
			return s, true
		}
	}
	return Scenario{}, false
}
