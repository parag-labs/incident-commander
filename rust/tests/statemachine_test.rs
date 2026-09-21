use incident_commander::models::{
    Alert, Incident, SERVICE_DATABASE, SEVERITY_CRITICAL, STATE_ESCALATED, STATE_INVESTIGATING,
    STATE_INVESTIGATION_FAILED, STATE_NEW, STATE_REMEDIATING, STATE_REMEDIATION_FAILED,
    STATE_REMEDIATION_PROPOSED, STATE_RESOLVED, STATE_ROOT_CAUSE_IDENTIFIED, STATE_TRIAGING,
    STATE_VERIFICATION_FAILED, STATE_VERIFYING, STATE_WAITING_FOR_APPROVAL,
};
use incident_commander::statemachine::{
    can_transition, is_terminal, new_incident, transition, IllegalTransition,
};

const EPOCH: i64 = 0;

fn new_test_incident() -> Incident {
    let alert = Alert {
        service: SERVICE_DATABASE.to_string(),
        metric: "error_rate".to_string(),
        severity: SEVERITY_CRITICAL.to_string(),
        ..Default::default()
    };
    new_incident("inc-1", alert, EPOCH)
}

#[test]
fn new_starts_in_new() {
    let inc = new_test_incident();
    assert_eq!(inc.state, STATE_NEW);
    assert_eq!(inc.history.len(), 1);
    assert_eq!(inc.history[0].to, STATE_NEW);
    assert_eq!(inc.history[0].from, "");
    assert_eq!(inc.history[0].reason, "incident opened");
}

#[test]
fn new_sets_title_from_alert() {
    assert_eq!(new_test_incident().title, "error_rate on database");
}

#[test]
fn happy_path_walks_to_resolved() {
    let mut inc = new_test_incident();
    let path = [
        STATE_TRIAGING,
        STATE_INVESTIGATING,
        STATE_ROOT_CAUSE_IDENTIFIED,
        STATE_REMEDIATION_PROPOSED,
        STATE_WAITING_FOR_APPROVAL,
        STATE_REMEDIATING,
        STATE_VERIFYING,
        STATE_RESOLVED,
    ];
    for to in path {
        transition(&mut inc, to, "step", EPOCH).unwrap();
    }
    assert_eq!(inc.state, STATE_RESOLVED);
    assert_eq!(inc.history.len(), 9);
}

#[test]
fn illegal_transition_is_rejected_and_does_not_mutate() {
    let mut inc = new_test_incident();
    let err = transition(&mut inc, STATE_REMEDIATING, "skip", EPOCH).unwrap_err();
    assert_eq!(
        err,
        IllegalTransition {
            from: STATE_NEW.to_string(),
            to: STATE_REMEDIATING.to_string(),
        }
    );
    assert_eq!(inc.state, STATE_NEW);
    assert_eq!(inc.history.len(), 1);
}

#[test]
fn illegal_transition_message() {
    let mut inc = new_test_incident();
    let err = transition(&mut inc, STATE_RESOLVED, "x", EPOCH).unwrap_err();
    assert_eq!(err.to_string(), "illegal transition NEW -> RESOLVED");
}

#[test]
fn terminal_states_have_no_exit() {
    for state in [
        STATE_RESOLVED,
        STATE_ESCALATED,
        STATE_INVESTIGATION_FAILED,
        STATE_REMEDIATION_FAILED,
        STATE_VERIFICATION_FAILED,
    ] {
        assert!(is_terminal(state));
        let mut inc = new_test_incident();
        inc.state = state.to_string();
        assert!(transition(&mut inc, STATE_RESOLVED, "x", EPOCH).is_err());
    }
}

#[test]
fn non_terminal_states_are_not_terminal() {
    for state in [
        STATE_NEW,
        STATE_TRIAGING,
        STATE_INVESTIGATING,
        STATE_ROOT_CAUSE_IDENTIFIED,
        STATE_REMEDIATION_PROPOSED,
        STATE_WAITING_FOR_APPROVAL,
        STATE_REMEDIATING,
        STATE_VERIFYING,
    ] {
        assert!(!is_terminal(state));
    }
}

#[test]
fn investigation_can_fail() {
    let mut inc = new_test_incident();
    transition(&mut inc, STATE_TRIAGING, "step", EPOCH).unwrap();
    transition(&mut inc, STATE_INVESTIGATING, "step", EPOCH).unwrap();
    transition(&mut inc, STATE_INVESTIGATION_FAILED, "no evidence", EPOCH).unwrap();
    assert!(is_terminal(&inc.state));
}

#[test]
fn read_only_resolution_skips_remediation() {
    let mut inc = new_test_incident();
    transition(&mut inc, STATE_TRIAGING, "step", EPOCH).unwrap();
    transition(&mut inc, STATE_INVESTIGATING, "step", EPOCH).unwrap();
    transition(&mut inc, STATE_ROOT_CAUSE_IDENTIFIED, "step", EPOCH).unwrap();
    transition(&mut inc, STATE_RESOLVED, "informational", EPOCH).unwrap();
    assert_eq!(inc.state, STATE_RESOLVED);
}

#[test]
fn can_transition_matches_table() {
    assert!(can_transition(STATE_NEW, STATE_TRIAGING));
    assert!(!can_transition(STATE_NEW, STATE_RESOLVED));
    assert!(can_transition(STATE_ROOT_CAUSE_IDENTIFIED, STATE_ESCALATED));
    assert!(!can_transition(STATE_RESOLVED, STATE_TRIAGING));
}

#[test]
fn can_transition_unknown_state_is_false() {
    assert!(!can_transition("BOGUS", STATE_TRIAGING));
}

#[test]
fn transition_records_from_and_reason() {
    let mut inc = new_test_incident();
    transition(&mut inc, STATE_TRIAGING, "begin triage", EPOCH).unwrap();
    let last = inc.history.last().unwrap();
    assert_eq!(last.from, STATE_NEW);
    assert_eq!(last.to, STATE_TRIAGING);
    assert_eq!(last.reason, "begin triage");
    assert_eq!(inc.updated_at, EPOCH);
}
