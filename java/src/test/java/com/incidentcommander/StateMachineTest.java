package com.incidentcommander;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.time.Instant;
import org.junit.jupiter.api.Test;

class StateMachineTest {
    private static final Instant EPOCH = Instant.EPOCH;

    private static Incident newTestIncident() {
        Alert alert = new Alert();
        alert.service = Services.DATABASE;
        alert.metric = "error_rate";
        alert.severity = Severity.CRITICAL;
        return StateMachine.newIncident("inc-1", alert, EPOCH);
    }

    @Test
    void newStartsInNew() {
        Incident inc = newTestIncident();
        assertEquals(States.NEW, inc.state);
        assertEquals(1, inc.history.size());
        assertEquals(States.NEW, inc.history.get(0).to);
        assertEquals("", inc.history.get(0).from);
        assertEquals("incident opened", inc.history.get(0).reason);
    }

    @Test
    void newSetsTitleFromAlert() {
        assertEquals("error_rate on database", newTestIncident().title);
    }

    @Test
    void happyPathWalksToResolved() {
        Incident inc = newTestIncident();
        String[] path = {
            States.TRIAGING,
            States.INVESTIGATING,
            States.ROOT_CAUSE_IDENTIFIED,
            States.REMEDIATION_PROPOSED,
            States.WAITING_FOR_APPROVAL,
            States.REMEDIATING,
            States.VERIFYING,
            States.RESOLVED,
        };
        for (String to : path) {
            StateMachine.transition(inc, to, "step", EPOCH);
        }
        assertEquals(States.RESOLVED, inc.state);
        assertEquals(9, inc.history.size());
    }

    @Test
    void illegalTransitionIsRejectedAndDoesNotMutate() {
        Incident inc = newTestIncident();
        IllegalTransitionException ex = assertThrows(
                IllegalTransitionException.class,
                () -> StateMachine.transition(inc, States.REMEDIATING, "skip", EPOCH));
        assertEquals(States.NEW, ex.getFrom());
        assertEquals(States.REMEDIATING, ex.getTo());
        assertEquals(States.NEW, inc.state);
        assertEquals(1, inc.history.size());
    }

    @Test
    void illegalTransitionMessage() {
        Incident inc = newTestIncident();
        IllegalTransitionException ex = assertThrows(
                IllegalTransitionException.class,
                () -> StateMachine.transition(inc, States.RESOLVED, "x", EPOCH));
        assertEquals("illegal transition NEW -> RESOLVED", ex.getMessage());
    }

    @Test
    void terminalStatesHaveNoExit() {
        String[] terminals = {
            States.RESOLVED,
            States.ESCALATED,
            States.INVESTIGATION_FAILED,
            States.REMEDIATION_FAILED,
            States.VERIFICATION_FAILED,
        };
        for (String state : terminals) {
            assertTrue(StateMachine.isTerminal(state));
            Incident inc = newTestIncident();
            inc.state = state;
            assertThrows(
                    IllegalTransitionException.class,
                    () -> StateMachine.transition(inc, States.RESOLVED, "x", EPOCH));
        }
    }

    @Test
    void nonTerminalStatesAreNotTerminal() {
        String[] states = {
            States.NEW,
            States.TRIAGING,
            States.INVESTIGATING,
            States.ROOT_CAUSE_IDENTIFIED,
            States.REMEDIATION_PROPOSED,
            States.WAITING_FOR_APPROVAL,
            States.REMEDIATING,
            States.VERIFYING,
        };
        for (String state : states) {
            assertFalse(StateMachine.isTerminal(state));
        }
    }

    @Test
    void investigationCanFail() {
        Incident inc = newTestIncident();
        StateMachine.transition(inc, States.TRIAGING, "step", EPOCH);
        StateMachine.transition(inc, States.INVESTIGATING, "step", EPOCH);
        StateMachine.transition(inc, States.INVESTIGATION_FAILED, "no evidence", EPOCH);
        assertTrue(StateMachine.isTerminal(inc.state));
    }

    @Test
    void readOnlyResolutionSkipsRemediation() {
        Incident inc = newTestIncident();
        StateMachine.transition(inc, States.TRIAGING, "step", EPOCH);
        StateMachine.transition(inc, States.INVESTIGATING, "step", EPOCH);
        StateMachine.transition(inc, States.ROOT_CAUSE_IDENTIFIED, "step", EPOCH);
        StateMachine.transition(inc, States.RESOLVED, "informational", EPOCH);
        assertEquals(States.RESOLVED, inc.state);
    }

    @Test
    void canTransitionMatchesTable() {
        assertTrue(StateMachine.canTransition(States.NEW, States.TRIAGING));
        assertFalse(StateMachine.canTransition(States.NEW, States.RESOLVED));
        assertTrue(StateMachine.canTransition(States.ROOT_CAUSE_IDENTIFIED, States.ESCALATED));
        assertFalse(StateMachine.canTransition(States.RESOLVED, States.TRIAGING));
    }

    @Test
    void canTransitionUnknownStateIsFalse() {
        assertFalse(StateMachine.canTransition("BOGUS", States.TRIAGING));
    }

    @Test
    void transitionRecordsFromAndReason() {
        Incident inc = newTestIncident();
        StateMachine.transition(inc, States.TRIAGING, "begin triage", EPOCH);
        Transition last = inc.history.get(inc.history.size() - 1);
        assertEquals(States.NEW, last.from);
        assertEquals(States.TRIAGING, last.to);
        assertEquals("begin triage", last.reason);
        assertEquals(EPOCH, inc.updatedAt);
    }
}
