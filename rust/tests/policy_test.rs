use incident_commander::models::{Remediation, SAFETY_HIGH, SAFETY_LOW, SAFETY_MEDIUM};
use incident_commander::policy::{assess, Config};

fn rem(action: &str) -> Remediation {
    Remediation {
        action: action.to_string(),
        ..Default::default()
    }
}

#[test]
fn low_always_auto_approvable() {
    let ra = assess(&rem("query_metrics"), SAFETY_LOW, Config::default());
    assert!(ra.auto_approvable);
    assert!(!ra.needs_approval);
    assert_eq!(ra.safety, SAFETY_LOW);
    assert_eq!(
        ra.reason,
        "read-only / low-risk action; safe to run automatically"
    );
}

#[test]
fn medium_needs_approval_when_disabled() {
    let ra = assess(
        &rem("restart_service"),
        SAFETY_MEDIUM,
        Config {
            auto_approve_medium: false,
        },
    );
    assert!(!ra.auto_approvable);
    assert!(ra.needs_approval);
    assert_eq!(
        ra.reason,
        "medium-risk action; requires human approval (auto-approval disabled)"
    );
}

#[test]
fn medium_auto_approvable_when_enabled() {
    let ra = assess(
        &rem("restart_service"),
        SAFETY_MEDIUM,
        Config {
            auto_approve_medium: true,
        },
    );
    assert!(ra.auto_approvable);
    assert!(!ra.needs_approval);
    assert_eq!(ra.reason, "medium-risk action; auto-approval is enabled");
}

#[test]
fn high_never_auto_approvable() {
    for enabled in [false, true] {
        let ra = assess(
            &rem("rollback_deployment"),
            SAFETY_HIGH,
            Config {
                auto_approve_medium: enabled,
            },
        );
        assert!(!ra.auto_approvable);
        assert!(ra.needs_approval);
        assert_eq!(
            ra.reason,
            "high-risk action; never runs without explicit human approval"
        );
    }
}

#[test]
fn unknown_safety_defaults_to_approval() {
    let ra = assess(
        &rem("???"),
        "weird",
        Config {
            auto_approve_medium: true,
        },
    );
    assert!(!ra.auto_approvable);
    assert!(ra.needs_approval);
    assert_eq!(
        ra.reason,
        "unknown safety level \"weird\"; defaulting to human approval"
    );
    assert_eq!(ra.safety, "weird");
}
