"""The deterministic risk gate between the AI's recommendation and any action.

The agent can recommend anything; nothing runs until this engine has classified it and
decided whether it may run automatically. This is the hard boundary that keeps
free-form model output from ever executing an infrastructure command on its own.
"""

from __future__ import annotations

from dataclasses import dataclass

from models import (
    SAFETY_HIGH,
    SAFETY_LOW,
    SAFETY_MEDIUM,
    Remediation,
    RiskAssessment,
    SafetyLevel,
)


@dataclass
class Config:
    """Controls how permissive the gate is.

    ``auto_approve_medium`` lets MEDIUM-safety remediations run without a human. LOW
    always may; HIGH never may, regardless of this flag.
    """

    auto_approve_medium: bool = False


def assess(rem: Remediation, safety: SafetyLevel, cfg: Config) -> RiskAssessment:
    """Classify a remediation and decide whether it may run automatically.

    The safety level is the tool's declared level (looked up by the caller), not
    anything the model asserts. ``rem`` is part of the audit contract but does not change
    the verdict, matching the reference.
    """
    del rem  # kept for signature parity with the reference; verdict depends on safety
    ra = RiskAssessment(safety=safety)
    if safety == SAFETY_LOW:
        ra.auto_approvable = True
        ra.reason = "read-only / low-risk action; safe to run automatically"
    elif safety == SAFETY_MEDIUM:
        if cfg.auto_approve_medium:
            ra.auto_approvable = True
            ra.reason = "medium-risk action; auto-approval is enabled"
        else:
            ra.needs_approval = True
            ra.reason = (
                "medium-risk action; requires human approval (auto-approval disabled)"
            )
    elif safety == SAFETY_HIGH:
        ra.needs_approval = True
        ra.reason = "high-risk action; never runs without explicit human approval"
    else:
        ra.needs_approval = True
        ra.reason = (
            f'unknown safety level "{safety}"; defaulting to human approval'
        )
    return ra
