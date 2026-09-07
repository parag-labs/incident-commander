// Package api is the HTTP surface: a small net/http server (no web framework) exposing
// incident creation and inspection, human approval, health, Prometheus metrics, and a
// minimal read-only MCP endpoint. Dangerous operations only ever run through the
// incident lifecycle and the policy gate - the API never exposes a raw "execute this
// tool" call.
package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/parag-labs/incident-commander/internal/agent"
	"github.com/parag-labs/incident-commander/internal/agent/llm"
	"github.com/parag-labs/incident-commander/internal/incident"
	"github.com/parag-labs/incident-commander/internal/obs"
	"github.com/parag-labs/incident-commander/internal/simulator"
	"github.com/parag-labs/incident-commander/internal/storage"
	"github.com/parag-labs/incident-commander/internal/tools"
	"github.com/parag-labs/incident-commander/pkg/models"
)

// Server holds the wiring for the HTTP API.
type Server struct {
	store    storage.Store
	runner   *agent.Runner
	registry *tools.Registry
	model    llm.LLMClient
	metrics  *obs.Metrics
	log      *slog.Logger

	mu   sync.Mutex
	envs map[string]*simulator.Env // incident ID -> its simulated environment
	seq  int
}

// Config configures a Server.
type Config struct {
	Model             llm.LLMClient
	Workers           int
	AutoApproveMedium bool
	Logger            *slog.Logger
}

// New builds a Server.
func New(cfg Config) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Model == nil {
		cfg.Model = llm.MockLLM{}
	}
	if cfg.Workers < 1 {
		cfg.Workers = 4
	}
	reg := tools.NewRegistry()
	return &Server{
		store:    storage.NewMemory(),
		runner:   agent.NewRunner(reg, cfg.Model, cfg.Workers),
		registry: reg,
		model:    cfg.Model,
		metrics:  obs.NewMetrics(),
		log:      cfg.Logger,
		envs:     map[string]*simulator.Env{},
	}
}

// Handler builds the HTTP routes.
func (s *Server) Handler(autoApproveMedium bool, workers int) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.health)
	mux.HandleFunc("GET /metrics", s.exposeMetrics)
	mux.HandleFunc("POST /incidents", s.createIncident(autoApproveMedium, workers))
	mux.HandleFunc("GET /incidents", s.listIncidents)
	mux.HandleFunc("GET /incidents/{id}", s.getIncident)
	mux.HandleFunc("GET /incidents/{id}/report", s.getReport)
	mux.HandleFunc("POST /incidents/{id}/approve", s.approveIncident)
	mux.HandleFunc("GET /scenarios", s.listScenarios)
	mux.HandleFunc("POST /mcp", s.mcp)
	return logging(s.log, mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) exposeMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = fmt.Fprint(w, s.metrics.Expose())
}

func (s *Server) listScenarios(w http.ResponseWriter, _ *http.Request) {
	type sc struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	out := make([]sc, 0, len(simulator.Scenarios))
	for _, x := range simulator.Scenarios {
		out = append(out, sc{ID: x.ID, Title: x.Title})
	}
	writeJSON(w, http.StatusOK, out)
}

type createRequest struct {
	Scenario          string `json:"scenario"`
	AutoApproveMedium *bool  `json:"auto_approve_medium,omitempty"`
}

func (s *Server) createIncident(defaultAuto bool, workers int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		sc, ok := simulator.ScenarioByID(req.Scenario)
		if !ok {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("unknown scenario %q", req.Scenario))
			return
		}
		auto := defaultAuto
		if req.AutoApproveMedium != nil {
			auto = *req.AutoApproveMedium
		}

		env := simulator.New(sc)
		id := s.nextID()
		inc := incident.New(id, sc.Alert, time.Now().UTC())

		res, err := s.runner.Run(r.Context(), env, inc, agent.Options{
			Workers: workers, AutoApproveMedium: auto, MaxToolCalls: 40,
		})
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.recordMetrics(res.Incident)
		s.store.Put(res.Incident)
		s.mu.Lock()
		s.envs[id] = env
		s.mu.Unlock()

		status := http.StatusCreated
		writeJSON(w, status, res.Incident)
	}
}

func (s *Server) listIncidents(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.List())
}

func (s *Server) getIncident(w http.ResponseWriter, r *http.Request) {
	inc, ok := s.store.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "incident not found")
		return
	}
	writeJSON(w, http.StatusOK, inc)
}

func (s *Server) getReport(w http.ResponseWriter, r *http.Request) {
	inc, ok := s.store.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "incident not found")
		return
	}
	report := agent.BuildReport(inc)
	narrative, _ := s.model.Explain(r.Context(), report)
	writeJSON(w, http.StatusOK, map[string]any{"report": report, "narrative": narrative})
}

func (s *Server) approveIncident(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inc, ok := s.store.Get(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "incident not found")
		return
	}
	s.mu.Lock()
	env := s.envs[id]
	s.mu.Unlock()
	if env == nil {
		writeErr(w, http.StatusConflict, "no environment for incident")
		return
	}
	res, err := s.runner.Approve(r.Context(), env, inc, agent.Options{})
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	s.recordMetrics(res.Incident)
	s.store.Put(res.Incident)
	writeJSON(w, http.StatusOK, res.Incident)
}

func (s *Server) recordMetrics(inc *models.Incident) {
	s.metrics.Inc("incident_count", 1)
	s.metrics.Inc("tool_call_count", float64(len(inc.ToolCalls)))
	if inc.State == models.StateResolved {
		s.metrics.Inc("incident_resolved_count", 1)
	}
	if inc.State == models.StateWaitingForApproval {
		s.metrics.Inc("human_approval_pending_count", 1)
	}
}

func (s *Server) nextID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return fmt.Sprintf("inc-%04d", s.seq)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func logging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info("request", "method", r.Method, "path", r.URL.Path, "dur_ms", time.Since(start).Milliseconds())
	})
}
