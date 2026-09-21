// The deterministic incident state machine, ported from the Go reference.
//
// This is the spine of the system: the LLM never mutates it. The orchestrator asks the
// machine to make a transition, the machine checks it is legal, records it, and refuses
// anything else.

import { Alert, Incident, State, States, Transition } from "./models.js";

// ALLOWED maps each state to the states it may legally move to. A transition not in this
// table is rejected - there is no path the machine will take that is not declared here.
const ALLOWED: Record<State, readonly State[]> = {
  [States.New]: [States.Triaging],
  [States.Triaging]: [States.Investigating, States.InvestigationFailed],
  [States.Investigating]: [States.RootCauseIdentified, States.InvestigationFailed],
  [States.RootCauseIdentified]: [States.RemediationProposed, States.Resolved, States.Escalated],
  [States.RemediationProposed]: [States.WaitingForApproval, States.Remediating, States.Escalated],
  [States.WaitingForApproval]: [States.Remediating, States.Escalated],
  [States.Remediating]: [States.Verifying, States.RemediationFailed],
  [States.Verifying]: [States.Resolved, States.VerificationFailed],
  [States.Resolved]: [],
  [States.InvestigationFailed]: [],
  [States.RemediationFailed]: [],
  [States.VerificationFailed]: [],
  [States.Escalated]: [],
};

// TERMINAL is the set of states from which the incident is finished.
const TERMINAL: ReadonlySet<State> = new Set([
  States.Resolved,
  States.InvestigationFailed,
  States.RemediationFailed,
  States.VerificationFailed,
  States.Escalated,
]);

/** Raised when a caller asks for a move the machine forbids. */
export class IllegalTransitionError extends Error {
  readonly from: State;
  readonly to: State;

  constructor(from: State, to: State) {
    super(`illegal transition ${from} -> ${to}`);
    this.name = "IllegalTransitionError";
    this.from = from;
    this.to = to;
  }
}

/** Report whether `from -> to` is a declared, legal move. */
export function canTransition(from: State, to: State): boolean {
  const targets = ALLOWED[from];
  return targets !== undefined && targets.includes(to);
}

/** Report whether an incident in this state is finished. */
export function isTerminal(state: State): boolean {
  return TERMINAL.has(state);
}

/** Create a fresh incident in the NEW state from an alert. */
export function newIncident(id: string, alert: Alert, now: Date): Incident {
  return new Incident({
    id,
    title: `${alert.metric} on ${alert.service}`,
    state: States.New,
    alert,
    createdAt: now,
    updatedAt: now,
    history: [
      new Transition({ from: "", to: States.New, reason: "incident opened", at: now }),
    ],
  });
}

/**
 * Advance the incident to a new state if the move is legal, recording it. Throws
 * {@link IllegalTransitionError} and leaves the incident untouched otherwise - the machine
 * never half-applies a move.
 */
export function transition(inc: Incident, to: State, reason: string, now: Date): void {
  if (!canTransition(inc.state, to)) {
    throw new IllegalTransitionError(inc.state, to);
  }
  inc.history.push(new Transition({ from: inc.state, to, reason, at: now }));
  inc.state = to;
  inc.updatedAt = now;
}
