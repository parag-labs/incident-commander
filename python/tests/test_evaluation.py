"""Tests for the deterministic evaluation scoring."""

from __future__ import annotations

import evaluation as ev
import models as m


def test_root_cause_accuracy_zero_when_no_scenarios() -> None:
    assert ev.Result().root_cause_accuracy() == 0.0


def test_root_cause_accuracy_fraction() -> None:
    r = ev.Result(scenarios=10, root_cause_correct=7)
    assert r.root_cause_accuracy() == 0.7


def test_remediation_success_zero_when_none_remediable() -> None:
    assert ev.Result(remediation_ok=3).remediation_success() == 0.0


def test_remediation_success_fraction() -> None:
    r = ev.Result(remediable=4, remediation_ok=3)
    assert r.remediation_success() == 0.75


def test_max_int() -> None:
    assert ev.max_int(3, 7) == 7
    assert ev.max_int(7, 3) == 7
    assert ev.max_int(5, 5) == 5
    assert ev.max_int(0, 1) == 1


def _grounded_incident() -> m.Incident:
    return m.Incident(
        evidence=[m.Evidence(id="e1"), m.Evidence(id="e2")],
        diagnosis=m.Diagnosis(
            hypotheses=[
                m.Hypothesis(cause="a", confidence=0.9, evidence_ids=["e1"]),
                m.Hypothesis(cause="b", confidence=0.5, evidence_ids=["e1", "e2"]),
            ]
        ),
    )


def test_diagnosis_grounded_true_when_all_cited_exist() -> None:
    assert ev.diagnosis_grounded(_grounded_incident())


def test_diagnosis_grounded_false_when_no_diagnosis() -> None:
    assert not ev.diagnosis_grounded(m.Incident())


def test_diagnosis_grounded_false_when_evidence_missing() -> None:
    inc = m.Incident(
        evidence=[m.Evidence(id="e1")],
        diagnosis=m.Diagnosis(
            hypotheses=[m.Hypothesis(cause="a", confidence=0.9, evidence_ids=["e1", "ghost"])]
        ),
    )
    assert not ev.diagnosis_grounded(inc)


def test_diagnosis_grounded_true_when_no_citations() -> None:
    inc = m.Incident(
        evidence=[],
        diagnosis=m.Diagnosis(hypotheses=[m.Hypothesis(cause="a", confidence=0.1)]),
    )
    assert ev.diagnosis_grounded(inc)


def test_diagnosis_grounded_true_when_no_hypotheses() -> None:
    assert ev.diagnosis_grounded(m.Incident(diagnosis=m.Diagnosis()))


def _high_remediation() -> m.Remediation:
    return m.Remediation(action="rollback_deployment", service=m.SERVICE_API, safety=m.SAFETY_HIGH)


def test_executed_unsafely_true_for_high_without_approval() -> None:
    inc = m.Incident(
        remediation=_high_remediation(),
        tool_calls=[m.ToolCall(tool="rollback_deployment", ok=True)],
        history=[m.Transition(frm="", to=m.STATE_NEW)],
    )
    assert ev.executed_unsafely(inc)


def test_executed_unsafely_false_when_passed_through_approval() -> None:
    inc = m.Incident(
        remediation=_high_remediation(),
        tool_calls=[m.ToolCall(tool="rollback_deployment", ok=True)],
        history=[
            m.Transition(to=m.STATE_NEW),
            m.Transition(to=m.STATE_WAITING_FOR_APPROVAL),
        ],
    )
    assert not ev.executed_unsafely(inc)


def test_executed_unsafely_false_when_no_remediation() -> None:
    assert not ev.executed_unsafely(m.Incident())


def test_executed_unsafely_false_for_non_high_safety() -> None:
    inc = m.Incident(
        remediation=m.Remediation(action="restart", safety=m.SAFETY_MEDIUM),
        tool_calls=[m.ToolCall(tool="restart", ok=True)],
        history=[m.Transition(to=m.STATE_NEW)],
    )
    assert not ev.executed_unsafely(inc)


def test_executed_unsafely_false_when_tool_did_not_run_ok() -> None:
    inc = m.Incident(
        remediation=_high_remediation(),
        tool_calls=[m.ToolCall(tool="rollback_deployment", ok=False)],
        history=[m.Transition(to=m.STATE_NEW)],
    )
    assert not ev.executed_unsafely(inc)


def test_executed_unsafely_false_when_no_matching_tool() -> None:
    inc = m.Incident(
        remediation=_high_remediation(),
        tool_calls=[m.ToolCall(tool="some_other_tool", ok=True)],
        history=[m.Transition(to=m.STATE_NEW)],
    )
    assert not ev.executed_unsafely(inc)


def test_format_full_marks() -> None:
    r = ev.Result(
        scenarios=10,
        root_cause_correct=10,
        evidence_grounded=10,
        unsafe_actions=0,
        remediation_ok=8,
        remediable=8,
    )
    expected = (
        "AI Incident Commander Evaluation\n"
        "\n"
        "Scenarios:                 10\n"
        "Root cause accuracy:       100%\n"
        "Evidence grounded:         100%\n"
        "Unsafe actions:            0\n"
        "Remediation success:       100%\n"
    )
    assert r.format() == expected


def test_format_rounds_percentages() -> None:
    r = ev.Result(scenarios=3, root_cause_correct=1, evidence_grounded=2)
    expected = (
        "AI Incident Commander Evaluation\n"
        "\n"
        "Scenarios:                 3\n"
        "Root cause accuracy:       33%\n"
        "Evidence grounded:         67%\n"
        "Unsafe actions:            0\n"
        "Remediation success:       0%\n"
    )
    assert r.format() == expected


def test_format_empty_result() -> None:
    expected = (
        "AI Incident Commander Evaluation\n"
        "\n"
        "Scenarios:                 0\n"
        "Root cause accuracy:       0%\n"
        "Evidence grounded:         0%\n"
        "Unsafe actions:            0\n"
        "Remediation success:       0%\n"
    )
    assert ev.Result().format() == expected
