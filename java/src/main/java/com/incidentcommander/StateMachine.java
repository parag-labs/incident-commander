package com.incidentcommander;

import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * The deterministic incident lifecycle.
 *
 * <p>This is the spine of the system: the LLM never mutates it. The orchestrator asks the
 * machine to make a transition, the machine checks it is legal, records it, and refuses
 * anything else.
 */
public final class StateMachine {
    private StateMachine() {
    }

    // ALLOWED maps each state to the states it may legally move to. A transition not in
    // this table is rejected - there is no path the machine will take that is not declared
    // here, so the flow cannot be steered somewhere unexpected by model output.
    private static final Map<String, List<String>> ALLOWED = Map.ofEntries(
            Map.entry(States.NEW, List.of(States.TRIAGING)),
            Map.entry(States.TRIAGING, List.of(States.INVESTIGATING, States.INVESTIGATION_FAILED)),
            Map.entry(
                    States.INVESTIGATING,
                    List.of(States.ROOT_CAUSE_IDENTIFIED, States.INVESTIGATION_FAILED)),
            Map.entry(
                    States.ROOT_CAUSE_IDENTIFIED,
                    List.of(States.REMEDIATION_PROPOSED, States.RESOLVED, States.ESCALATED)),
            Map.entry(
                    States.REMEDIATION_PROPOSED,
                    List.of(States.WAITING_FOR_APPROVAL, States.REMEDIATING, States.ESCALATED)),
            Map.entry(
                    States.WAITING_FOR_APPROVAL,
                    List.of(States.REMEDIATING, States.ESCALATED)),
            Map.entry(States.REMEDIATING, List.of(States.VERIFYING, States.REMEDIATION_FAILED)),
            Map.entry(States.VERIFYING, List.of(States.RESOLVED, States.VERIFICATION_FAILED)),
            Map.entry(States.RESOLVED, List.of()),
            Map.entry(States.INVESTIGATION_FAILED, List.of()),
            Map.entry(States.REMEDIATION_FAILED, List.of()),
            Map.entry(States.VERIFICATION_FAILED, List.of()),
            Map.entry(States.ESCALATED, List.of()));

    // TERMINAL is the set of states from which the incident is finished.
    private static final Set<String> TERMINAL = Set.of(
            States.RESOLVED,
            States.INVESTIGATION_FAILED,
            States.REMEDIATION_FAILED,
            States.VERIFICATION_FAILED,
            States.ESCALATED);

    /**
     * Report whether {@code from -> to} is a declared, legal move.
     *
     * @param from the source state
     * @param to the destination state
     * @return whether the move is legal
     */
    public static boolean canTransition(String from, String to) {
        return ALLOWED.getOrDefault(from, List.of()).contains(to);
    }

    /**
     * Report whether an incident in this state is finished.
     *
     * @param state the state to test
     * @return whether the state is terminal
     */
    public static boolean isTerminal(String state) {
        return TERMINAL.contains(state);
    }

    /**
     * Create a fresh incident in the {@code NEW} state from an alert.
     *
     * @param id the incident identifier
     * @param alert the alert that opened it
     * @param now the creation timestamp
     * @return the new incident
     */
    public static Incident newIncident(String id, Alert alert, Instant now) {
        Incident inc = new Incident();
        inc.id = id;
        inc.title = alert.metric + " on " + alert.service;
        inc.state = States.NEW;
        inc.alert = alert;
        inc.createdAt = now;
        inc.updatedAt = now;
        Transition opening = new Transition();
        opening.from = "";
        opening.to = States.NEW;
        opening.reason = "incident opened";
        opening.at = now;
        inc.history.add(opening);
        return inc;
    }

    /**
     * Advance the incident to a new state if the move is legal, recording it. Throws
     * {@link IllegalTransitionException} and leaves the incident untouched otherwise - the
     * machine never half-applies a move.
     *
     * @param inc the incident to advance
     * @param to the destination state
     * @param reason why the move happens
     * @param now the transition timestamp
     */
    public static void transition(Incident inc, String to, String reason, Instant now) {
        if (!canTransition(inc.state, to)) {
            throw new IllegalTransitionException(inc.state, to);
        }
        Transition step = new Transition();
        step.from = inc.state;
        step.to = to;
        step.reason = reason;
        step.at = now;
        inc.history.add(step);
        inc.state = to;
        inc.updatedAt = now;
    }
}
