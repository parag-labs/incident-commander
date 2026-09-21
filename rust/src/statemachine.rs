//! The deterministic incident state machine.
//!
//! This is the spine of the system: the LLM never mutates it. The orchestrator asks the
//! machine to make a transition, the machine checks it is legal, records it, and refuses
//! anything else.

use std::error::Error;
use std::fmt;

use crate::models::{
    Alert, Incident, Transition, STATE_ESCALATED, STATE_INVESTIGATING, STATE_INVESTIGATION_FAILED,
    STATE_NEW, STATE_REMEDIATING, STATE_REMEDIATION_FAILED, STATE_REMEDIATION_PROPOSED,
    STATE_RESOLVED, STATE_ROOT_CAUSE_IDENTIFIED, STATE_TRIAGING, STATE_VERIFICATION_FAILED,
    STATE_VERIFYING, STATE_WAITING_FOR_APPROVAL,
};

/// Return the states `from` may legally move to. An unknown or terminal state yields an
/// empty slice, so no transition out of it is ever allowed.
fn allowed_targets(from: &str) -> &'static [&'static str] {
    if from == STATE_NEW {
        &[STATE_TRIAGING]
    } else if from == STATE_TRIAGING {
        &[STATE_INVESTIGATING, STATE_INVESTIGATION_FAILED]
    } else if from == STATE_INVESTIGATING {
        &[STATE_ROOT_CAUSE_IDENTIFIED, STATE_INVESTIGATION_FAILED]
    } else if from == STATE_ROOT_CAUSE_IDENTIFIED {
        &[STATE_REMEDIATION_PROPOSED, STATE_RESOLVED, STATE_ESCALATED]
    } else if from == STATE_REMEDIATION_PROPOSED {
        &[
            STATE_WAITING_FOR_APPROVAL,
            STATE_REMEDIATING,
            STATE_ESCALATED,
        ]
    } else if from == STATE_WAITING_FOR_APPROVAL {
        &[STATE_REMEDIATING, STATE_ESCALATED]
    } else if from == STATE_REMEDIATING {
        &[STATE_VERIFYING, STATE_REMEDIATION_FAILED]
    } else if from == STATE_VERIFYING {
        &[STATE_RESOLVED, STATE_VERIFICATION_FAILED]
    } else {
        &[]
    }
}

/// Report whether an incident in this state is finished.
pub fn is_terminal(state: &str) -> bool {
    state == STATE_RESOLVED
        || state == STATE_INVESTIGATION_FAILED
        || state == STATE_REMEDIATION_FAILED
        || state == STATE_VERIFICATION_FAILED
        || state == STATE_ESCALATED
}

/// Report whether `from -> to` is a declared, legal move.
pub fn can_transition(from: &str, to: &str) -> bool {
    allowed_targets(from).contains(&to)
}

/// Returned when a caller asks for a move the machine forbids.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct IllegalTransition {
    /// The state the move started from.
    pub from: String,
    /// The rejected destination state.
    pub to: String,
}

impl fmt::Display for IllegalTransition {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "illegal transition {} -> {}", self.from, self.to)
    }
}

impl Error for IllegalTransition {}

/// Create a fresh incident in the `NEW` state from an alert.
pub fn new_incident(id: &str, alert: Alert, now: i64) -> Incident {
    let title = format!("{} on {}", alert.metric, alert.service);
    Incident {
        id: id.to_string(),
        title,
        state: STATE_NEW.to_string(),
        alert,
        created_at: now,
        updated_at: now,
        history: vec![Transition {
            from: String::new(),
            to: STATE_NEW.to_string(),
            reason: "incident opened".to_string(),
            at: now,
        }],
        ..Default::default()
    }
}

/// Advance the incident to a new state if the move is legal, recording it.
///
/// Returns [`IllegalTransition`] and leaves the incident untouched otherwise - the
/// machine never half-applies a move.
pub fn transition(
    inc: &mut Incident,
    to: &str,
    reason: &str,
    now: i64,
) -> Result<(), IllegalTransition> {
    if !can_transition(&inc.state, to) {
        return Err(IllegalTransition {
            from: inc.state.clone(),
            to: to.to_string(),
        });
    }
    inc.history.push(Transition {
        from: inc.state.clone(),
        to: to.to_string(),
        reason: reason.to_string(),
        at: now,
    });
    inc.state = to.to_string();
    inc.updated_at = now;
    Ok(())
}
