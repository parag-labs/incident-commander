// Package llm is the boundary between the deterministic system and the language model.
// The rest of the app depends only on the LLMClient interface, never on a provider, so
// the same agent runs against a real model in production and a deterministic mock in
// CI. The contract is deliberately narrow: the model returns *text* (a JSON diagnosis),
// and the agent parses and validates it - the model never returns anything the
// deterministic layer executes without checking.
package llm

import (
	"context"

	"github.com/parag-labs/incident-commander/pkg/models"
)

// DeployNote is a recent deployment fact passed to the model as evidence.
type DeployNote struct {
	Service    models.Service `json:"service"`
	Version    string         `json:"version"`
	MinutesAgo int            `json:"minutes_ago"`
}

// EvidenceBundle is everything the investigator has gathered, handed to the model as
// the only ground truth it may reason from. There is nothing free-form here - the model
// must cite evidence IDs that appear in Evidence.
type EvidenceBundle struct {
	IncidentID  string            `json:"incident_id"`
	Alert       models.Alert      `json:"alert"`
	Evidence    []models.Evidence `json:"evidence"`
	Deployments []DeployNote      `json:"deployments"`
}

// LLMClient is the single provider-agnostic interface the agent depends on. A mock
// implementation is required for tests; an OpenAI-compatible one ships for production.
type LLMClient interface {
	// Name identifies the provider for logs and metrics.
	Name() string
	// Diagnose returns the model's raw structured response (a JSON models.Diagnosis).
	// It intentionally returns text, not a parsed struct, so the agent owns parsing and
	// validation - that is where a malformed or hallucinated response is caught.
	Diagnose(ctx context.Context, bundle EvidenceBundle) (raw string, err error)
	// Explain turns a finished incident into a human-readable narrative.
	Explain(ctx context.Context, report models.IncidentReport) (string, error)
}
