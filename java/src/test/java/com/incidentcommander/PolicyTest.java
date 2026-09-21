package com.incidentcommander;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import org.junit.jupiter.api.Test;

class PolicyTest {
    private static Remediation rem(String action) {
        Remediation r = new Remediation();
        r.action = action;
        return r;
    }

    @Test
    void lowAlwaysAutoApprovable() {
        RiskAssessment ra = Policy.assess(rem("query_metrics"), Safety.LOW, new Policy.Config());
        assertTrue(ra.autoApprovable);
        assertFalse(ra.needsApproval);
        assertEquals(Safety.LOW, ra.safety);
        assertEquals("read-only / low-risk action; safe to run automatically", ra.reason);
    }

    @Test
    void mediumNeedsApprovalWhenDisabled() {
        RiskAssessment ra = Policy.assess(rem("restart_service"), Safety.MEDIUM, new Policy.Config(false));
        assertFalse(ra.autoApprovable);
        assertTrue(ra.needsApproval);
        assertEquals(
                "medium-risk action; requires human approval (auto-approval disabled)",
                ra.reason);
    }

    @Test
    void mediumAutoApprovableWhenEnabled() {
        RiskAssessment ra = Policy.assess(rem("restart_service"), Safety.MEDIUM, new Policy.Config(true));
        assertTrue(ra.autoApprovable);
        assertFalse(ra.needsApproval);
        assertEquals("medium-risk action; auto-approval is enabled", ra.reason);
    }

    @Test
    void highNeverAutoApprovable() {
        for (boolean enabled : new boolean[] {false, true}) {
            RiskAssessment ra =
                    Policy.assess(rem("rollback_deployment"), Safety.HIGH, new Policy.Config(enabled));
            assertFalse(ra.autoApprovable);
            assertTrue(ra.needsApproval);
            assertEquals("high-risk action; never runs without explicit human approval", ra.reason);
        }
    }

    @Test
    void unknownSafetyDefaultsToApproval() {
        RiskAssessment ra = Policy.assess(rem("???"), "weird", new Policy.Config(true));
        assertFalse(ra.autoApprovable);
        assertTrue(ra.needsApproval);
        assertEquals("unknown safety level \"weird\"; defaulting to human approval", ra.reason);
        assertEquals("weird", ra.safety);
    }
}
