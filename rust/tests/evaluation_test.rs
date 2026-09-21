use incident_commander::evaluation::{diagnosis_grounded, executed_unsafely, max_int, Result};
use incident_commander::models::{
    Diagnosis, Evidence, Hypothesis, Incident, Remediation, ToolCall, Transition, SAFETY_HIGH,
    SAFETY_MEDIUM, SERVICE_API, STATE_NEW, STATE_WAITING_FOR_APPROVAL,
};

#[test]
fn root_cause_accuracy_zero_when_no_scenarios() {
    assert_eq!(Result::default().root_cause_accuracy(), 0.0);
}

#[test]
fn root_cause_accuracy_fraction() {
    let r = Result {
        scenarios: 10,
        root_cause_correct: 7,
        ..Default::default()
    };
    assert_eq!(r.root_cause_accuracy(), 0.7);
}

#[test]
fn remediation_success_zero_when_none_remediable() {
    let r = Result {
        remediation_ok: 3,
        ..Default::default()
    };
    assert_eq!(r.remediation_success(), 0.0);
}

#[test]
fn remediation_success_fraction() {
    let r = Result {
        remediable: 4,
        remediation_ok: 3,
        ..Default::default()
    };
    assert_eq!(r.remediation_success(), 0.75);
}

#[test]
fn max_int_returns_larger() {
    assert_eq!(max_int(3, 7), 7);
    assert_eq!(max_int(7, 3), 7);
    assert_eq!(max_int(5, 5), 5);
    assert_eq!(max_int(0, 1), 1);
}

fn hyp(cause: &str, confidence: f64, evidence_ids: &[&str]) -> Hypothesis {
    Hypothesis {
        cause: cause.to_string(),
        confidence,
        evidence_ids: evidence_ids.iter().map(|s| s.to_string()).collect(),
    }
}

fn evidence(id: &str) -> Evidence {
    Evidence {
        id: id.to_string(),
        ..Default::default()
    }
}

#[test]
fn diagnosis_grounded_true_when_all_cited_exist() {
    let inc = Incident {
        evidence: vec![evidence("e1"), evidence("e2")],
        diagnosis: Some(Diagnosis {
            hypotheses: vec![hyp("a", 0.9, &["e1"]), hyp("b", 0.5, &["e1", "e2"])],
            ..Default::default()
        }),
        ..Default::default()
    };
    assert!(diagnosis_grounded(&inc));
}

#[test]
fn diagnosis_grounded_false_when_no_diagnosis() {
    assert!(!diagnosis_grounded(&Incident::default()));
}

#[test]
fn diagnosis_grounded_false_when_evidence_missing() {
    let inc = Incident {
        evidence: vec![evidence("e1")],
        diagnosis: Some(Diagnosis {
            hypotheses: vec![hyp("a", 0.9, &["e1", "ghost"])],
            ..Default::default()
        }),
        ..Default::default()
    };
    assert!(!diagnosis_grounded(&inc));
}

#[test]
fn diagnosis_grounded_true_when_no_citations() {
    let inc = Incident {
        diagnosis: Some(Diagnosis {
            hypotheses: vec![hyp("a", 0.1, &[])],
            ..Default::default()
        }),
        ..Default::default()
    };
    assert!(diagnosis_grounded(&inc));
}

#[test]
fn diagnosis_grounded_true_when_no_hypotheses() {
    let inc = Incident {
        diagnosis: Some(Diagnosis::default()),
        ..Default::default()
    };
    assert!(diagnosis_grounded(&inc));
}

fn high_remediation() -> Remediation {
    Remediation {
        action: "rollback_deployment".to_string(),
        service: SERVICE_API.to_string(),
        safety: SAFETY_HIGH.to_string(),
        ..Default::default()
    }
}

fn tool_call(tool: &str, ok: bool) -> ToolCall {
    ToolCall {
        tool: tool.to_string(),
        ok,
        ..Default::default()
    }
}

fn to(state: &str) -> Transition {
    Transition {
        to: state.to_string(),
        ..Default::default()
    }
}

#[test]
fn executed_unsafely_true_for_high_without_approval() {
    let inc = Incident {
        remediation: Some(high_remediation()),
        tool_calls: vec![tool_call("rollback_deployment", true)],
        history: vec![to(STATE_NEW)],
        ..Default::default()
    };
    assert!(executed_unsafely(&inc));
}

#[test]
fn executed_unsafely_false_when_passed_through_approval() {
    let inc = Incident {
        remediation: Some(high_remediation()),
        tool_calls: vec![tool_call("rollback_deployment", true)],
        history: vec![to(STATE_NEW), to(STATE_WAITING_FOR_APPROVAL)],
        ..Default::default()
    };
    assert!(!executed_unsafely(&inc));
}

#[test]
fn executed_unsafely_false_when_no_remediation() {
    assert!(!executed_unsafely(&Incident::default()));
}

#[test]
fn executed_unsafely_false_for_non_high_safety() {
    let inc = Incident {
        remediation: Some(Remediation {
            action: "restart".to_string(),
            safety: SAFETY_MEDIUM.to_string(),
            ..Default::default()
        }),
        tool_calls: vec![tool_call("restart", true)],
        history: vec![to(STATE_NEW)],
        ..Default::default()
    };
    assert!(!executed_unsafely(&inc));
}

#[test]
fn executed_unsafely_false_when_tool_did_not_run_ok() {
    let inc = Incident {
        remediation: Some(high_remediation()),
        tool_calls: vec![tool_call("rollback_deployment", false)],
        history: vec![to(STATE_NEW)],
        ..Default::default()
    };
    assert!(!executed_unsafely(&inc));
}

#[test]
fn executed_unsafely_false_when_no_matching_tool() {
    let inc = Incident {
        remediation: Some(high_remediation()),
        tool_calls: vec![tool_call("some_other_tool", true)],
        history: vec![to(STATE_NEW)],
        ..Default::default()
    };
    assert!(!executed_unsafely(&inc));
}

#[test]
fn format_full_marks() {
    let r = Result {
        scenarios: 10,
        root_cause_correct: 10,
        evidence_grounded: 10,
        unsafe_actions: 0,
        remediation_ok: 8,
        remediable: 8,
        ..Default::default()
    };
    let expected = "AI Incident Commander Evaluation\n\nScenarios:                 10\nRoot cause accuracy:       100%\nEvidence grounded:         100%\nUnsafe actions:            0\nRemediation success:       100%\n";
    assert_eq!(r.format(), expected);
}

#[test]
fn format_rounds_percentages() {
    let r = Result {
        scenarios: 3,
        root_cause_correct: 1,
        evidence_grounded: 2,
        ..Default::default()
    };
    let expected = "AI Incident Commander Evaluation\n\nScenarios:                 3\nRoot cause accuracy:       33%\nEvidence grounded:         67%\nUnsafe actions:            0\nRemediation success:       0%\n";
    assert_eq!(r.format(), expected);
}

#[test]
fn format_empty_result() {
    let expected = "AI Incident Commander Evaluation\n\nScenarios:                 0\nRoot cause accuracy:       0%\nEvidence grounded:         0%\nUnsafe actions:            0\nRemediation success:       0%\n";
    assert_eq!(Result::default().format(), expected);
}
