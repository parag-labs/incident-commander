"""Tests for the domain models (constants and Diagnosis.top)."""

from __future__ import annotations

import models as m


def test_all_services_stable_order() -> None:
    assert m.ALL_SERVICES == ["api", "payment", "database", "cache", "queue"]


def test_state_constants_are_readable_strings() -> None:
    assert m.STATE_NEW == "NEW"
    assert m.STATE_WAITING_FOR_APPROVAL == "WAITING_FOR_APPROVAL"
    assert m.SAFETY_LOW == "low"
    assert m.SEVERITY_CRITICAL == "critical"


def test_diagnosis_top_returns_highest_confidence() -> None:
    d = m.Diagnosis(
        hypotheses=[
            m.Hypothesis(cause="a", confidence=0.3),
            m.Hypothesis(cause="b", confidence=0.9),
            m.Hypothesis(cause="c", confidence=0.5),
        ]
    )
    top = d.top()
    assert top is not None
    assert top.cause == "b"


def test_diagnosis_top_ties_keep_first() -> None:
    d = m.Diagnosis(
        hypotheses=[
            m.Hypothesis(cause="first", confidence=0.8),
            m.Hypothesis(cause="second", confidence=0.8),
        ]
    )
    top = d.top()
    assert top is not None
    assert top.cause == "first"


def test_diagnosis_top_empty_is_none() -> None:
    assert m.Diagnosis().top() is None


def test_diagnosis_top_handles_zero_confidence() -> None:
    d = m.Diagnosis(hypotheses=[m.Hypothesis(cause="only", confidence=0.0)])
    top = d.top()
    assert top is not None
    assert top.cause == "only"
