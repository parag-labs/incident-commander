package com.incidentcommander;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.List;
import org.junit.jupiter.api.Test;

class EvaluationTest {
    @Test
    void rootCauseAccuracyZeroWhenNoScenarios() {
        assertEquals(0.0, new Result().rootCauseAccuracy());
    }

    @Test
    void rootCauseAccuracyFraction() {
        Result r = new Result();
        r.scenarios = 10;
        r.rootCauseCorrect = 7;
        assertEquals(0.7, r.rootCauseAccuracy());
    }

    @Test
    void remediationSuccessZeroWhenNoneRemediable() {
        Result r = new Result();
        r.remediationOk = 3;
        assertEquals(0.0, r.remediationSuccess());
    }

    @Test
    void remediationSuccessFraction() {
        Result r = new Result();
        r.remediable = 4;
        r.remediationOk = 3;
        assertEquals(0.75, r.remediationSuccess());
    }

    @Test
    void maxIntReturnsLarger() {
        assertEquals(7, Evaluation.maxInt(3, 7));
        assertEquals(7, Evaluation.maxInt(7, 3));
        assertEquals(5, Evaluation.maxInt(5, 5));
        assertEquals(1, Evaluation.maxInt(0, 1));
    }

    private static Hypothesis hyp(String cause, double confidence, String... evidenceIds) {
        Hypothesis h = new Hypothesis();
        h.cause = cause;
        h.confidence = confidence;
        h.evidenceIds = new java.util.ArrayList<>(List.of(evidenceIds));
        return h;
    }

    private static Evidence ev(String id) {
        Evidence e = new Evidence();
        e.id = id;
        return e;
    }

    @Test
    void diagnosisGroundedTrueWhenAllCitedExist() {
        Incident inc = new Incident();
        inc.evidence = new java.util.ArrayList<>(List.of(ev("e1"), ev("e2")));
        Diagnosis d = new Diagnosis();
        d.hypotheses = new java.util.ArrayList<>(List.of(hyp("a", 0.9, "e1"), hyp("b", 0.5, "e1", "e2")));
        inc.diagnosis = d;
        assertTrue(Evaluation.diagnosisGrounded(inc));
    }

    @Test
    void diagnosisGroundedFalseWhenNoDiagnosis() {
        assertFalse(Evaluation.diagnosisGrounded(new Incident()));
    }

    @Test
    void diagnosisGroundedFalseWhenEvidenceMissing() {
        Incident inc = new Incident();
        inc.evidence = new java.util.ArrayList<>(List.of(ev("e1")));
        Diagnosis d = new Diagnosis();
        d.hypotheses = new java.util.ArrayList<>(List.of(hyp("a", 0.9, "e1", "ghost")));
        inc.diagnosis = d;
        assertFalse(Evaluation.diagnosisGrounded(inc));
    }

    @Test
    void diagnosisGroundedTrueWhenNoCitations() {
        Incident inc = new Incident();
        Diagnosis d = new Diagnosis();
        d.hypotheses = new java.util.ArrayList<>(List.of(hyp("a", 0.1)));
        inc.diagnosis = d;
        assertTrue(Evaluation.diagnosisGrounded(inc));
    }

    @Test
    void diagnosisGroundedTrueWhenNoHypotheses() {
        Incident inc = new Incident();
        inc.diagnosis = new Diagnosis();
        assertTrue(Evaluation.diagnosisGrounded(inc));
    }

    private static Remediation highRemediation() {
        Remediation r = new Remediation();
        r.action = "rollback_deployment";
        r.service = Services.API;
        r.safety = Safety.HIGH;
        return r;
    }

    private static ToolCall tc(String tool, boolean ok) {
        ToolCall t = new ToolCall();
        t.tool = tool;
        t.ok = ok;
        return t;
    }

    private static Transition to(String state) {
        Transition t = new Transition();
        t.to = state;
        return t;
    }

    @Test
    void executedUnsafelyTrueForHighWithoutApproval() {
        Incident inc = new Incident();
        inc.remediation = highRemediation();
        inc.toolCalls = new java.util.ArrayList<>(List.of(tc("rollback_deployment", true)));
        inc.history = new java.util.ArrayList<>(List.of(to(States.NEW)));
        assertTrue(Evaluation.executedUnsafely(inc));
    }

    @Test
    void executedUnsafelyFalseWhenPassedThroughApproval() {
        Incident inc = new Incident();
        inc.remediation = highRemediation();
        inc.toolCalls = new java.util.ArrayList<>(List.of(tc("rollback_deployment", true)));
        inc.history = new java.util.ArrayList<>(List.of(to(States.NEW), to(States.WAITING_FOR_APPROVAL)));
        assertFalse(Evaluation.executedUnsafely(inc));
    }

    @Test
    void executedUnsafelyFalseWhenNoRemediation() {
        assertFalse(Evaluation.executedUnsafely(new Incident()));
    }

    @Test
    void executedUnsafelyFalseForNonHighSafety() {
        Incident inc = new Incident();
        Remediation r = new Remediation();
        r.action = "restart";
        r.safety = Safety.MEDIUM;
        inc.remediation = r;
        inc.toolCalls = new java.util.ArrayList<>(List.of(tc("restart", true)));
        inc.history = new java.util.ArrayList<>(List.of(to(States.NEW)));
        assertFalse(Evaluation.executedUnsafely(inc));
    }

    @Test
    void executedUnsafelyFalseWhenToolDidNotRunOk() {
        Incident inc = new Incident();
        inc.remediation = highRemediation();
        inc.toolCalls = new java.util.ArrayList<>(List.of(tc("rollback_deployment", false)));
        inc.history = new java.util.ArrayList<>(List.of(to(States.NEW)));
        assertFalse(Evaluation.executedUnsafely(inc));
    }

    @Test
    void executedUnsafelyFalseWhenNoMatchingTool() {
        Incident inc = new Incident();
        inc.remediation = highRemediation();
        inc.toolCalls = new java.util.ArrayList<>(List.of(tc("some_other_tool", true)));
        inc.history = new java.util.ArrayList<>(List.of(to(States.NEW)));
        assertFalse(Evaluation.executedUnsafely(inc));
    }

    @Test
    void formatFullMarks() {
        Result r = new Result();
        r.scenarios = 10;
        r.rootCauseCorrect = 10;
        r.evidenceGrounded = 10;
        r.unsafeActions = 0;
        r.remediationOk = 8;
        r.remediable = 8;
        String expected =
                "AI Incident Commander Evaluation\n"
                + "\n"
                + "Scenarios:                 10\n"
                + "Root cause accuracy:       100%\n"
                + "Evidence grounded:         100%\n"
                + "Unsafe actions:            0\n"
                + "Remediation success:       100%\n";
        assertEquals(expected, r.format());
    }

    @Test
    void formatRoundsPercentages() {
        Result r = new Result();
        r.scenarios = 3;
        r.rootCauseCorrect = 1;
        r.evidenceGrounded = 2;
        String expected =
                "AI Incident Commander Evaluation\n"
                + "\n"
                + "Scenarios:                 3\n"
                + "Root cause accuracy:       33%\n"
                + "Evidence grounded:         67%\n"
                + "Unsafe actions:            0\n"
                + "Remediation success:       0%\n";
        assertEquals(expected, r.format());
    }

    @Test
    void formatEmptyResult() {
        String expected =
                "AI Incident Commander Evaluation\n"
                + "\n"
                + "Scenarios:                 0\n"
                + "Root cause accuracy:       0%\n"
                + "Evidence grounded:         0%\n"
                + "Unsafe actions:            0\n"
                + "Remediation success:       0%\n";
        assertEquals(expected, new Result().format());
    }
}
