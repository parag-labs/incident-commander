package agent

import (
	"testing"

	"github.com/parag-labs/incident-commander/internal/tools"
	"github.com/parag-labs/incident-commander/pkg/models"
)

func evidenceSet() []models.Evidence {
	return []models.Evidence{{ID: "e1", Service: models.ServiceDatabase}, {ID: "e2", Service: models.ServiceAPI}}
}

func TestValidateAcceptsAGoodDiagnosis(t *testing.T) {
	raw := `{"summary":"db exhausted","hypotheses":[{"cause":"pool full","confidence":0.9,"evidence_ids":["e1"]}],
	"recommended_action":"reduce_db_connections","recommended_service":"database","risk":"medium","needs_human_approval":false}`
	d, err := Validate(raw, evidenceSet(), tools.NewRegistry())
	if err != nil {
		t.Fatalf("valid diagnosis rejected: %v", err)
	}
	if d.RecommendedAction != "reduce_db_connections" {
		t.Fatalf("unexpected action %q", d.RecommendedAction)
	}
}

func TestValidateRejectsMalformedJSON(t *testing.T) {
	_, err := Validate(`{not valid json`, evidenceSet(), tools.NewRegistry())
	if _, ok := err.(ErrMalformed); !ok {
		t.Fatalf("want ErrMalformed, got %v", err)
	}
}

func TestValidateRejectsHallucinatedTool(t *testing.T) {
	raw := `{"summary":"x","recommended_action":"delete_production_database","recommended_service":"database","risk":"high"}`
	_, err := Validate(raw, evidenceSet(), tools.NewRegistry())
	if _, ok := err.(ErrHallucinatedTool); !ok {
		t.Fatalf("want ErrHallucinatedTool, got %v", err)
	}
}

func TestValidateRejectsUnknownEvidence(t *testing.T) {
	raw := `{"summary":"x","hypotheses":[{"cause":"c","confidence":0.5,"evidence_ids":["e999"]}]}`
	_, err := Validate(raw, evidenceSet(), tools.NewRegistry())
	if _, ok := err.(ErrUnknownEvidence); !ok {
		t.Fatalf("want ErrUnknownEvidence, got %v", err)
	}
}

func TestValidateRejectsOutOfRangeConfidence(t *testing.T) {
	raw := `{"summary":"x","hypotheses":[{"cause":"c","confidence":1.7,"evidence_ids":["e1"]}]}`
	_, err := Validate(raw, evidenceSet(), tools.NewRegistry())
	if _, ok := err.(ErrInvalidConfidence); !ok {
		t.Fatalf("want ErrInvalidConfidence, got %v", err)
	}
}

func TestValidateRejectsUnsafeServiceForAction(t *testing.T) {
	raw := `{"summary":"x","recommended_action":"restart_service","recommended_service":"not_a_service","risk":"medium"}`
	_, err := Validate(raw, evidenceSet(), tools.NewRegistry())
	if _, ok := err.(ErrInvalidService); !ok {
		t.Fatalf("want ErrInvalidService, got %v", err)
	}
}

func TestValidateAllowsEmptyActionForInformational(t *testing.T) {
	raw := `{"summary":"all healthy","recommended_action":"","risk":"low","needs_human_approval":true}`
	if _, err := Validate(raw, evidenceSet(), tools.NewRegistry()); err != nil {
		t.Fatalf("an informational (no-action) diagnosis should be valid, got %v", err)
	}
}
