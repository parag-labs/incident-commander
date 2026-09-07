// Package agent orchestrates an investigation: it collects evidence concurrently, asks
// the model for a diagnosis, validates that diagnosis hard, and only then lets the
// deterministic layer act. The model proposes; this package makes sure nothing it
// proposes is trusted without checking it against reality.
package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/parag-labs/incident-commander/pkg/models"
)

// Validation errors are typed so tests (and metrics) can distinguish exactly how a
// model response was rejected.
type (
	// ErrMalformed means the model returned text that isn't the expected JSON shape.
	ErrMalformed struct{ Cause error }
	// ErrHallucinatedTool means the model recommended an action that is not a real tool.
	ErrHallucinatedTool struct{ Action string }
	// ErrUnknownEvidence means a hypothesis cited an evidence ID that was never collected.
	ErrUnknownEvidence struct{ ID string }
	// ErrInvalidConfidence means a confidence was outside [0,1].
	ErrInvalidConfidence struct{ Value float64 }
	// ErrInvalidService means the recommended service isn't a real service.
	ErrInvalidService struct{ Service models.Service }
	// ErrInvalidRisk means the risk wasn't one of low/medium/high.
	ErrInvalidRisk struct{ Risk models.Risk }
)

func (e ErrMalformed) Error() string { return fmt.Sprintf("malformed model output: %v", e.Cause) }
func (e ErrHallucinatedTool) Error() string {
	return fmt.Sprintf("model recommended a non-existent tool %q", e.Action)
}
func (e ErrUnknownEvidence) Error() string {
	return fmt.Sprintf("model cited unknown evidence id %q", e.ID)
}
func (e ErrInvalidConfidence) Error() string {
	return fmt.Sprintf("confidence %.3f out of range [0,1]", e.Value)
}
func (e ErrInvalidService) Error() string {
	return fmt.Sprintf("recommended service %q is not a known service", e.Service)
}
func (e ErrInvalidRisk) Error() string { return fmt.Sprintf("invalid risk %q", e.Risk) }

// toolChecker reports whether a name is a real remediation tool.
type toolChecker interface {
	IsRemediator(name string) bool
}

// Validate parses and hard-checks a raw model diagnosis against the ground truth of
// what was actually collected and what tools actually exist. A valid Diagnosis is safe
// for the deterministic layer to act on; anything else is rejected with a typed error.
func Validate(raw string, evidence []models.Evidence, tools toolChecker) (models.Diagnosis, error) {
	var d models.Diagnosis
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return models.Diagnosis{}, ErrMalformed{Cause: err}
	}

	known := map[string]bool{}
	for _, e := range evidence {
		known[e.ID] = true
	}

	for _, h := range d.Hypotheses {
		if h.Confidence < 0 || h.Confidence > 1 {
			return models.Diagnosis{}, ErrInvalidConfidence{Value: h.Confidence}
		}
		for _, id := range h.EvidenceIDs {
			if !known[id] {
				return models.Diagnosis{}, ErrUnknownEvidence{ID: id}
			}
		}
	}

	switch d.Risk {
	case models.RiskLow, models.RiskMedium, models.RiskHigh, "":
		// ok
	default:
		return models.Diagnosis{}, ErrInvalidRisk{Risk: d.Risk}
	}

	// An empty action is legitimate (a read-only / informational incident). A non-empty
	// action must name a real remediation tool and a real service.
	if d.RecommendedAction != "" {
		if !tools.IsRemediator(d.RecommendedAction) {
			return models.Diagnosis{}, ErrHallucinatedTool{Action: d.RecommendedAction}
		}
		if !validService(d.RecommendedService) {
			return models.Diagnosis{}, ErrInvalidService{Service: d.RecommendedService}
		}
	}

	return d, nil
}

func validService(s models.Service) bool {
	for _, x := range models.AllServices {
		if x == s {
			return true
		}
	}
	return false
}
