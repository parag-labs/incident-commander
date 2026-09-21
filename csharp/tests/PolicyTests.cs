using Xunit;

namespace IncidentCommander.Tests
{
    public class PolicyTests
    {
        private static Remediation Rem(string action)
        {
            return new Remediation { Action = action };
        }

        [Fact]
        public void LowAlwaysAutoApprovable()
        {
            RiskAssessment ra = Policy.Assess(Rem("query_metrics"), Safety.Low, new Policy.Config());
            Assert.True(ra.AutoApprovable);
            Assert.False(ra.NeedsApproval);
            Assert.Equal(Safety.Low, ra.SafetyLevel);
            Assert.Equal("read-only / low-risk action; safe to run automatically", ra.Reason);
        }

        [Fact]
        public void MediumNeedsApprovalWhenDisabled()
        {
            RiskAssessment ra = Policy.Assess(
                Rem("restart_service"),
                Safety.Medium,
                new Policy.Config { AutoApproveMedium = false });
            Assert.False(ra.AutoApprovable);
            Assert.True(ra.NeedsApproval);
            Assert.Equal(
                "medium-risk action; requires human approval (auto-approval disabled)",
                ra.Reason);
        }

        [Fact]
        public void MediumAutoApprovableWhenEnabled()
        {
            RiskAssessment ra = Policy.Assess(
                Rem("restart_service"),
                Safety.Medium,
                new Policy.Config { AutoApproveMedium = true });
            Assert.True(ra.AutoApprovable);
            Assert.False(ra.NeedsApproval);
            Assert.Equal("medium-risk action; auto-approval is enabled", ra.Reason);
        }

        [Fact]
        public void HighNeverAutoApprovable()
        {
            foreach (bool enabled in new[] { false, true })
            {
                RiskAssessment ra = Policy.Assess(
                    Rem("rollback_deployment"),
                    Safety.High,
                    new Policy.Config { AutoApproveMedium = enabled });
                Assert.False(ra.AutoApprovable);
                Assert.True(ra.NeedsApproval);
                Assert.Equal(
                    "high-risk action; never runs without explicit human approval",
                    ra.Reason);
            }
        }

        [Fact]
        public void UnknownSafetyDefaultsToApproval()
        {
            RiskAssessment ra = Policy.Assess(
                Rem("???"),
                "weird",
                new Policy.Config { AutoApproveMedium = true });
            Assert.False(ra.AutoApprovable);
            Assert.True(ra.NeedsApproval);
            Assert.Equal("unknown safety level \"weird\"; defaulting to human approval", ra.Reason);
            Assert.Equal("weird", ra.SafetyLevel);
        }
    }
}
