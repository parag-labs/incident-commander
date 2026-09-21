use incident_commander::models::{
    Diagnosis, Hypothesis, ALL_SERVICES, SAFETY_LOW, SEVERITY_CRITICAL, STATE_NEW,
    STATE_WAITING_FOR_APPROVAL,
};

#[test]
fn all_services_stable_order() {
    assert_eq!(
        ALL_SERVICES,
        ["api", "payment", "database", "cache", "queue"]
    );
}

#[test]
fn state_constants_are_readable_strings() {
    assert_eq!(STATE_NEW, "NEW");
    assert_eq!(STATE_WAITING_FOR_APPROVAL, "WAITING_FOR_APPROVAL");
    assert_eq!(SAFETY_LOW, "low");
    assert_eq!(SEVERITY_CRITICAL, "critical");
}

fn hyp(cause: &str, confidence: f64) -> Hypothesis {
    Hypothesis {
        cause: cause.to_string(),
        confidence,
        ..Default::default()
    }
}

#[test]
fn diagnosis_top_returns_highest_confidence() {
    let d = Diagnosis {
        hypotheses: vec![hyp("a", 0.3), hyp("b", 0.9), hyp("c", 0.5)],
        ..Default::default()
    };
    assert_eq!(d.top().unwrap().cause, "b");
}

#[test]
fn diagnosis_top_ties_keep_first() {
    let d = Diagnosis {
        hypotheses: vec![hyp("first", 0.8), hyp("second", 0.8)],
        ..Default::default()
    };
    assert_eq!(d.top().unwrap().cause, "first");
}

#[test]
fn diagnosis_top_empty_is_none() {
    assert!(Diagnosis::default().top().is_none());
}

#[test]
fn diagnosis_top_handles_zero_confidence() {
    let d = Diagnosis {
        hypotheses: vec![hyp("only", 0.0)],
        ..Default::default()
    };
    assert_eq!(d.top().unwrap().cause, "only");
}
