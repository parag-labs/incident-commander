package llm

import (
	"context"

	"github.com/parag-labs/incident-commander/pkg/models"
)

// ScriptedLLM returns a fixed raw response (or error), letting tests drive the agent's
// validation layer with exactly the adversarial model output they want to exercise:
// malformed JSON, a hallucinated tool name, an unsafe action, missing evidence IDs, or
// an out-of-range confidence. It is the deterministic "bad model" for the AI tests.
type ScriptedLLM struct {
	Raw     string
	Err     error
	Narrate string
}

// Name identifies the provider.
func (ScriptedLLM) Name() string { return "scripted" }

// Diagnose returns the scripted raw response.
func (s ScriptedLLM) Diagnose(_ context.Context, _ EvidenceBundle) (string, error) {
	if s.Err != nil {
		return "", s.Err
	}
	return s.Raw, nil
}

// Explain returns the scripted narrative.
func (s ScriptedLLM) Explain(_ context.Context, _ models.IncidentReport) (string, error) {
	return s.Narrate, nil
}
