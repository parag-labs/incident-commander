// Package incident holds the deterministic incident state machine. This is the spine
// of the system: the LLM never mutates it. The orchestrator asks the machine to make
// a transition, the machine checks it's legal, records it, and refuses anything else.
// Keeping this deterministic and separate from the model is what lets an
// AI-in-the-loop system stay auditable.
package incident

import (
	"fmt"
	"time"

	"github.com/parag-labs/incident-commander/pkg/models"
)

// allowed maps each state to the states it may legally move to. A transition not in
// this table is rejected - there is no path the machine will take that isn't declared
// here, so the flow can't be steered somewhere unexpected by model output.
var allowed = map[models.State][]models.State{
	models.StateNew:                 {models.StateTriaging},
	models.StateTriaging:            {models.StateInvestigating, models.StateInvestigationFailed},
	models.StateInvestigating:       {models.StateRootCauseIdentified, models.StateInvestigationFailed},
	models.StateRootCauseIdentified: {models.StateRemediationProposed, models.StateResolved, models.StateEscalated},
	models.StateRemediationProposed: {models.StateWaitingForApproval, models.StateRemediating, models.StateEscalated},
	models.StateWaitingForApproval:  {models.StateRemediating, models.StateEscalated},
	models.StateRemediating:         {models.StateVerifying, models.StateRemediationFailed},
	models.StateVerifying:           {models.StateResolved, models.StateVerificationFailed},

	// Terminal / failure states have no outgoing transitions.
	models.StateResolved:            {},
	models.StateInvestigationFailed: {},
	models.StateRemediationFailed:   {},
	models.StateVerificationFailed:  {},
	models.StateEscalated:           {},
}

// terminal is the set of states from which the incident is finished.
var terminal = map[models.State]bool{
	models.StateResolved:            true,
	models.StateInvestigationFailed: true,
	models.StateRemediationFailed:   true,
	models.StateVerificationFailed:  true,
	models.StateEscalated:           true,
}

// ErrIllegalTransition is returned when a caller asks for a move the machine forbids.
type ErrIllegalTransition struct {
	From models.State
	To   models.State
}

func (e ErrIllegalTransition) Error() string {
	return fmt.Sprintf("illegal transition %s -> %s", e.From, e.To)
}

// CanTransition reports whether from -> to is a declared, legal move.
func CanTransition(from, to models.State) bool {
	for _, s := range allowed[from] {
		if s == to {
			return true
		}
	}
	return false
}

// IsTerminal reports whether an incident in this state is finished.
func IsTerminal(s models.State) bool { return terminal[s] }

// New creates a fresh incident in the NEW state from an alert.
func New(id string, alert models.Alert, now time.Time) *models.Incident {
	title := fmt.Sprintf("%s on %s", alert.Metric, alert.Service)
	return &models.Incident{
		ID:        id,
		Title:     title,
		State:     models.StateNew,
		Alert:     alert,
		CreatedAt: now,
		UpdatedAt: now,
		History: []models.Transition{
			{From: "", To: models.StateNew, Reason: "incident opened", At: now},
		},
	}
}

// Transition advances the incident to a new state if the move is legal, recording it
// in the history. It returns ErrIllegalTransition and leaves the incident untouched
// otherwise - the machine never half-applies a move.
func Transition(inc *models.Incident, to models.State, reason string, now time.Time) error {
	if !CanTransition(inc.State, to) {
		return ErrIllegalTransition{From: inc.State, To: to}
	}
	inc.History = append(inc.History, models.Transition{
		From: inc.State, To: to, Reason: reason, At: now,
	})
	inc.State = to
	inc.UpdatedAt = now
	return nil
}
