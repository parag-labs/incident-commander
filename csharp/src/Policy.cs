// The deterministic risk gate between the AI's recommendation and any action.
//
// The agent can recommend anything; nothing runs until this engine has classified it and
// decided whether it may run automatically.

using System;

namespace IncidentCommander
{
    /// <summary>The risk-gating policy engine.</summary>
    public static class Policy
    {
        /// <summary>Controls how permissive the gate is.</summary>
        public sealed class Config
        {
            /// <summary>Lets MEDIUM-safety remediations run without a human. LOW always
            /// may; HIGH never may, regardless of this flag.</summary>
            public bool AutoApproveMedium { get; set; }
        }

        /// <summary>Classify a remediation and decide whether it may run automatically.
        /// The safety level is the tool's declared level (looked up by the caller), not
        /// anything the model asserts. <paramref name="rem"/> is part of the audit
        /// contract but does not change the verdict, matching the reference.</summary>
        public static RiskAssessment Assess(Remediation rem, string safety, Config cfg)
        {
            _ = rem;
            RiskAssessment ra = new RiskAssessment { SafetyLevel = safety };
            switch (safety)
            {
                case Safety.Low:
                    ra.AutoApprovable = true;
                    ra.Reason = "read-only / low-risk action; safe to run automatically";
                    break;
                case Safety.Medium:
                    if (cfg.AutoApproveMedium)
                    {
                        ra.AutoApprovable = true;
                        ra.Reason = "medium-risk action; auto-approval is enabled";
                    }
                    else
                    {
                        ra.NeedsApproval = true;
                        ra.Reason =
                            "medium-risk action; requires human approval (auto-approval disabled)";
                    }

                    break;
                case Safety.High:
                    ra.NeedsApproval = true;
                    ra.Reason = "high-risk action; never runs without explicit human approval";
                    break;
                default:
                    ra.NeedsApproval = true;
                    ra.Reason = $"unknown safety level \"{safety}\"; defaulting to human approval";
                    break;
            }

            return ra;
        }
    }
}
