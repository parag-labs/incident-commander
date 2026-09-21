"""The deterministic incident state machine.

This is the spine of the system: the LLM never mutates it. The orchestrator asks the
machine to make a transition, the machine checks it is legal, records it, and refuses
anything else. Keeping this deterministic and separate from the model is what lets an
AI-in-the-loop system stay auditable.
"""

from __future__ import annotations

from datetime import datetime

from models import (
    STATE_ESCALATED,
    STATE_INVESTIGATING,
    STATE_INVESTIGATION_FAILED,
    STATE_NEW,
    STATE_REMEDIATING,
    STATE_REMEDIATION_FAILED,
    STATE_REMEDIATION_PROPOSED,
    STATE_RESOLVED,
    STATE_ROOT_CAUSE_IDENTIFIED,
    STATE_TRIAGING,
    STATE_VERIFICATION_FAILED,
    STATE_VERIFYING,
    STATE_WAITING_FOR_APPROVAL,
    Alert,
    Incident,
    State,
    Transition,
)

# ALLOWED maps each state to the states it may legally move to. A transition not in
# this table is rejected - there is no path the machine will take that is not declared
# here, so the flow cannot be steered somewhere unexpected by model output.
ALLOWED: dict[State, list[State]] = {
    STATE_NEW: [STATE_TRIAGING],
    STATE_TRIAGING: [STATE_INVESTIGATING, STATE_INVESTIGATION_FAILED],
    STATE_INVESTIGATING: [STATE_ROOT_CAUSE_IDENTIFIED, STATE_INVESTIGATION_FAILED],
    STATE_ROOT_CAUSE_IDENTIFIED: [
        STATE_REMEDIATION_PROPOSED,
        STATE_RESOLVED,
        STATE_ESCALATED,
    ],
    STATE_REMEDIATION_PROPOSED: [
        STATE_WAITING_FOR_APPROVAL,
        STATE_REMEDIATING,
        STATE_ESCALATED,
    ],
    STATE_WAITING_FOR_APPROVAL: [STATE_REMEDIATING, STATE_ESCALATED],
    STATE_REMEDIATING: [STATE_VERIFYING, STATE_REMEDIATION_FAILED],
    STATE_VERIFYING: [STATE_RESOLVED, STATE_VERIFICATION_FAILED],
    # Terminal / failure states have no outgoing transitions.
    STATE_RESOLVED: [],
    STATE_INVESTIGATION_FAILED: [],
    STATE_REMEDIATION_FAILED: [],
    STATE_VERIFICATION_FAILED: [],
    STATE_ESCALATED: [],
}

# TERMINAL is the set of states from which the incident is finished.
TERMINAL: set[State] = {
    STATE_RESOLVED,
    STATE_INVESTIGATION_FAILED,
    STATE_REMEDIATION_FAILED,
    STATE_VERIFICATION_FAILED,
    STATE_ESCALATED,
}


class IllegalTransitionError(Exception):
    """Raised when a caller asks for a move the machine forbids."""

    def __init__(self, frm: State, to: State) -> None:
        self.frm = frm
        self.to = to
        super().__init__(f"illegal transition {frm} -> {to}")


def can_transition(frm: State, to: State) -> bool:
    """Report whether ``frm -> to`` is a declared, legal move."""
    return to in ALLOWED.get(frm, [])


def is_terminal(state: State) -> bool:
    """Report whether an incident in this state is finished."""
    return state in TERMINAL


def new_incident(incident_id: str, alert: Alert, now: datetime) -> Incident:
    """Create a fresh incident in the NEW state from an alert."""
    title = f"{alert.metric} on {alert.service}"
    return Incident(
        id=incident_id,
        title=title,
        state=STATE_NEW,
        alert=alert,
        created_at=now,
        updated_at=now,
        history=[Transition(frm="", to=STATE_NEW, reason="incident opened", at=now)],
    )


def transition(incident: Incident, to: State, reason: str, now: datetime) -> None:
    """Advance the incident to a new state if the move is legal, recording it.

    Raises :class:`IllegalTransitionError` and leaves the incident untouched otherwise -
    the machine never half-applies a move.
    """
    if not can_transition(incident.state, to):
        raise IllegalTransitionError(incident.state, to)
    incident.history.append(
        Transition(frm=incident.state, to=to, reason=reason, at=now)
    )
    incident.state = to
    incident.updated_at = now
