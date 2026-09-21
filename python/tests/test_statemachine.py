"""Tests for the incident state machine (mirrors the Go statemachine_test.go)."""

from __future__ import annotations

from datetime import datetime

import pytest

import models as m
import statemachine as sm

EPOCH = datetime(1970, 1, 1)


def new_test_incident() -> m.Incident:
    return sm.new_incident(
        "inc-1",
        m.Alert(
            service=m.SERVICE_DATABASE,
            metric="error_rate",
            severity=m.SEVERITY_CRITICAL,
        ),
        EPOCH,
    )


def test_new_starts_in_new() -> None:
    inc = new_test_incident()
    assert inc.state == m.STATE_NEW
    assert len(inc.history) == 1
    assert inc.history[0].to == m.STATE_NEW
    assert inc.history[0].frm == ""
    assert inc.history[0].reason == "incident opened"


def test_new_sets_title_from_alert() -> None:
    inc = new_test_incident()
    assert inc.title == "error_rate on database"


def test_happy_path_walks_to_resolved() -> None:
    inc = new_test_incident()
    path = [
        m.STATE_TRIAGING,
        m.STATE_INVESTIGATING,
        m.STATE_ROOT_CAUSE_IDENTIFIED,
        m.STATE_REMEDIATION_PROPOSED,
        m.STATE_WAITING_FOR_APPROVAL,
        m.STATE_REMEDIATING,
        m.STATE_VERIFYING,
        m.STATE_RESOLVED,
    ]
    for to in path:
        sm.transition(inc, to, "step", EPOCH)
    assert inc.state == m.STATE_RESOLVED
    # opening entry + 8 transitions
    assert len(inc.history) == 9


def test_illegal_transition_is_rejected_and_does_not_mutate() -> None:
    inc = new_test_incident()
    with pytest.raises(sm.IllegalTransitionError) as excinfo:
        sm.transition(inc, m.STATE_REMEDIATING, "skip", EPOCH)
    assert excinfo.value.frm == m.STATE_NEW
    assert excinfo.value.to == m.STATE_REMEDIATING
    assert inc.state == m.STATE_NEW
    assert len(inc.history) == 1


def test_illegal_transition_message() -> None:
    inc = new_test_incident()
    with pytest.raises(sm.IllegalTransitionError) as excinfo:
        sm.transition(inc, m.STATE_RESOLVED, "x", EPOCH)
    assert str(excinfo.value) == "illegal transition NEW -> RESOLVED"


def test_terminal_states_have_no_exit() -> None:
    for state in [
        m.STATE_RESOLVED,
        m.STATE_ESCALATED,
        m.STATE_INVESTIGATION_FAILED,
        m.STATE_REMEDIATION_FAILED,
        m.STATE_VERIFICATION_FAILED,
    ]:
        assert sm.is_terminal(state)
        inc = new_test_incident()
        inc.state = state
        with pytest.raises(sm.IllegalTransitionError):
            sm.transition(inc, m.STATE_RESOLVED, "x", EPOCH)


def test_non_terminal_states_are_not_terminal() -> None:
    for state in [
        m.STATE_NEW,
        m.STATE_TRIAGING,
        m.STATE_INVESTIGATING,
        m.STATE_ROOT_CAUSE_IDENTIFIED,
        m.STATE_REMEDIATION_PROPOSED,
        m.STATE_WAITING_FOR_APPROVAL,
        m.STATE_REMEDIATING,
        m.STATE_VERIFYING,
    ]:
        assert not sm.is_terminal(state)


def test_investigation_can_fail() -> None:
    inc = new_test_incident()
    sm.transition(inc, m.STATE_TRIAGING, "step", EPOCH)
    sm.transition(inc, m.STATE_INVESTIGATING, "step", EPOCH)
    sm.transition(inc, m.STATE_INVESTIGATION_FAILED, "no evidence", EPOCH)
    assert sm.is_terminal(inc.state)


def test_read_only_resolution_skips_remediation() -> None:
    inc = new_test_incident()
    sm.transition(inc, m.STATE_TRIAGING, "step", EPOCH)
    sm.transition(inc, m.STATE_INVESTIGATING, "step", EPOCH)
    sm.transition(inc, m.STATE_ROOT_CAUSE_IDENTIFIED, "step", EPOCH)
    sm.transition(inc, m.STATE_RESOLVED, "informational", EPOCH)
    assert inc.state == m.STATE_RESOLVED


def test_can_transition_matches_table() -> None:
    assert sm.can_transition(m.STATE_NEW, m.STATE_TRIAGING)
    assert not sm.can_transition(m.STATE_NEW, m.STATE_RESOLVED)
    assert sm.can_transition(m.STATE_ROOT_CAUSE_IDENTIFIED, m.STATE_ESCALATED)
    assert not sm.can_transition(m.STATE_RESOLVED, m.STATE_TRIAGING)


def test_can_transition_unknown_state_is_false() -> None:
    assert not sm.can_transition("BOGUS", m.STATE_TRIAGING)


def test_transition_records_from_and_reason() -> None:
    inc = new_test_incident()
    sm.transition(inc, m.STATE_TRIAGING, "begin triage", EPOCH)
    last = inc.history[-1]
    assert last.frm == m.STATE_NEW
    assert last.to == m.STATE_TRIAGING
    assert last.reason == "begin triage"
    assert inc.updated_at == EPOCH
