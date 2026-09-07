package api

import (
	"encoding/json"
	"net/http"

	"github.com/parag-labs/incident-commander/internal/simulator"
	"github.com/parag-labs/incident-commander/pkg/models"
)

// mcpRequest is a minimal MCP-style tool call: a method name and JSON params. This is a
// deliberately small surface that exposes only READ-ONLY investigation capabilities.
// Remediation is never reachable here - dangerous actions only run through the incident
// lifecycle and the policy gate.
type mcpRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// mcp handles the read-only MCP endpoint. Supported methods:
//
//	service.health   {"incident":"inc-0001","service":"database"}
//	metrics.query    {"incident":"inc-0001","service":"api"}
//	deployment.list  {"incident":"inc-0001"}
//	incident.get     {"incident":"inc-0001"}
func (s *Server) mcp(w http.ResponseWriter, r *http.Request) {
	var req mcpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid MCP request")
		return
	}
	var p struct {
		Incident string         `json:"incident"`
		Service  models.Service `json:"service"`
	}
	_ = json.Unmarshal(req.Params, &p)

	env := s.envFor(p.Incident)

	switch req.Method {
	case "service.health":
		if env == nil {
			writeErr(w, http.StatusNotFound, "unknown incident environment")
			return
		}
		writeJSON(w, http.StatusOK, env.Snapshot(p.Service))
	case "metrics.query":
		if env == nil {
			writeErr(w, http.StatusNotFound, "unknown incident environment")
			return
		}
		writeJSON(w, http.StatusOK, env.All())
	case "deployment.list":
		if env == nil {
			writeErr(w, http.StatusNotFound, "unknown incident environment")
			return
		}
		writeJSON(w, http.StatusOK, env.Deployments())
	case "incident.get":
		inc, ok := s.store.Get(p.Incident)
		if !ok {
			writeErr(w, http.StatusNotFound, "incident not found")
			return
		}
		writeJSON(w, http.StatusOK, inc)
	default:
		writeErr(w, http.StatusBadRequest, "unsupported or non-read-only MCP method: "+req.Method)
	}
}

func (s *Server) envFor(id string) *simulator.Env {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.envs[id]
}
