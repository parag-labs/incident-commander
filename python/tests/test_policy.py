"""Tests for the deterministic policy gate (mirrors the Go policy_test.go)."""

from __future__ import annotations

import models as m
import policy


def test_low_always_auto_approvable() -> None:
    ra = policy.assess(m.Remediation(action="query_metrics"), m.SAFETY_LOW, policy.Config())
    assert ra.auto_approvable
    assert not ra.needs_approval
    assert ra.safety == m.SAFETY_LOW
    assert ra.reason == "read-only / low-risk action; safe to run automatically"


def test_medium_needs_approval_when_disabled() -> None:
    ra = policy.assess(
        m.Remediation(action="restart_service"),
        m.SAFETY_MEDIUM,
        policy.Config(auto_approve_medium=False),
    )
    assert not ra.auto_approvable
    assert ra.needs_approval
    assert ra.reason == (
        "medium-risk action; requires human approval (auto-approval disabled)"
    )


def test_medium_auto_approvable_when_enabled() -> None:
    ra = policy.assess(
        m.Remediation(action="restart_service"),
        m.SAFETY_MEDIUM,
        policy.Config(auto_approve_medium=True),
    )
    assert ra.auto_approvable
    assert not ra.needs_approval
    assert ra.reason == "medium-risk action; auto-approval is enabled"


def test_high_never_auto_approvable() -> None:
    for cfg in [policy.Config(auto_approve_medium=False), policy.Config(auto_approve_medium=True)]:
        ra = policy.assess(m.Remediation(action="rollback_deployment"), m.SAFETY_HIGH, cfg)
        assert not ra.auto_approvable
        assert ra.needs_approval
        assert ra.reason == "high-risk action; never runs without explicit human approval"


def test_unknown_safety_defaults_to_approval() -> None:
    ra = policy.assess(
        m.Remediation(action="???"),
        "weird",
        policy.Config(auto_approve_medium=True),
    )
    assert not ra.auto_approvable
    assert ra.needs_approval
    assert ra.reason == 'unknown safety level "weird"; defaulting to human approval'
    assert ra.safety == "weird"
