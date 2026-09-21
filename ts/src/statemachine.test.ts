import { describe, expect, it } from "vitest";
import { Alert, Incident, Services, Severities, States } from "./models.js";
import {
  IllegalTransitionError,
  canTransition,
  isTerminal,
  newIncident,
  transition,
} from "./statemachine.js";

const EPOCH = new Date(0);

function newTestIncident(): Incident {
  const alert = new Alert({
    service: Services.Database,
    metric: "error_rate",
    severity: Severities.Critical,
  });
  return newIncident("inc-1", alert, EPOCH);
}

describe("statemachine", () => {
  it("starts in NEW with an opening history entry", () => {
    const inc = newTestIncident();
    expect(inc.state).toBe(States.New);
    expect(inc.history).toHaveLength(1);
    expect(inc.history[0].to).toBe(States.New);
    expect(inc.history[0].from).toBe("");
    expect(inc.history[0].reason).toBe("incident opened");
  });

  it("sets the title from the alert", () => {
    expect(newTestIncident().title).toBe("error_rate on database");
  });

  it("walks the happy path to RESOLVED", () => {
    const inc = newTestIncident();
    const path = [
      States.Triaging,
      States.Investigating,
      States.RootCauseIdentified,
      States.RemediationProposed,
      States.WaitingForApproval,
      States.Remediating,
      States.Verifying,
      States.Resolved,
    ];
    for (const to of path) {
      transition(inc, to, "step", EPOCH);
    }
    expect(inc.state).toBe(States.Resolved);
    expect(inc.history).toHaveLength(9);
  });

  it("rejects an illegal transition and does not mutate", () => {
    const inc = newTestIncident();
    expect(() => transition(inc, States.Remediating, "skip", EPOCH)).toThrow(
      IllegalTransitionError,
    );
    expect(inc.state).toBe(States.New);
    expect(inc.history).toHaveLength(1);
  });

  it("reports from/to on the illegal transition error", () => {
    const inc = newTestIncident();
    try {
      transition(inc, States.Resolved, "x", EPOCH);
      expect.unreachable();
    } catch (err) {
      expect(err).toBeInstanceOf(IllegalTransitionError);
      const e = err as IllegalTransitionError;
      expect(e.from).toBe(States.New);
      expect(e.to).toBe(States.Resolved);
      expect(e.message).toBe("illegal transition NEW -> RESOLVED");
    }
  });

  it("treats terminal states as having no exit", () => {
    const terminals = [
      States.Resolved,
      States.Escalated,
      States.InvestigationFailed,
      States.RemediationFailed,
      States.VerificationFailed,
    ];
    for (const state of terminals) {
      expect(isTerminal(state)).toBe(true);
      const inc = newTestIncident();
      inc.state = state;
      expect(() => transition(inc, States.Resolved, "x", EPOCH)).toThrow(IllegalTransitionError);
    }
  });

  it("treats active states as non-terminal", () => {
    const states = [
      States.New,
      States.Triaging,
      States.Investigating,
      States.RootCauseIdentified,
      States.RemediationProposed,
      States.WaitingForApproval,
      States.Remediating,
      States.Verifying,
    ];
    for (const state of states) {
      expect(isTerminal(state)).toBe(false);
    }
  });

  it("allows investigation to fail", () => {
    const inc = newTestIncident();
    transition(inc, States.Triaging, "step", EPOCH);
    transition(inc, States.Investigating, "step", EPOCH);
    transition(inc, States.InvestigationFailed, "no evidence", EPOCH);
    expect(isTerminal(inc.state)).toBe(true);
  });

  it("allows a read-only resolution that skips remediation", () => {
    const inc = newTestIncident();
    transition(inc, States.Triaging, "step", EPOCH);
    transition(inc, States.Investigating, "step", EPOCH);
    transition(inc, States.RootCauseIdentified, "step", EPOCH);
    transition(inc, States.Resolved, "informational", EPOCH);
    expect(inc.state).toBe(States.Resolved);
  });

  it("matches the transition table", () => {
    expect(canTransition(States.New, States.Triaging)).toBe(true);
    expect(canTransition(States.New, States.Resolved)).toBe(false);
    expect(canTransition(States.RootCauseIdentified, States.Escalated)).toBe(true);
    expect(canTransition(States.Resolved, States.Triaging)).toBe(false);
  });

  it("returns false for an unknown source state", () => {
    expect(canTransition("BOGUS", States.Triaging)).toBe(false);
  });

  it("records from and reason on a transition", () => {
    const inc = newTestIncident();
    transition(inc, States.Triaging, "begin triage", EPOCH);
    const last = inc.history[inc.history.length - 1];
    expect(last.from).toBe(States.New);
    expect(last.to).toBe(States.Triaging);
    expect(last.reason).toBe("begin triage");
    expect(inc.updatedAt).toBe(EPOCH);
  });
});
