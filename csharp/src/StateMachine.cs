// The deterministic incident state machine.
//
// This is the spine of the system: the LLM never mutates it. The orchestrator asks the
// machine to make a transition, the machine checks it is legal, records it, and refuses
// anything else.

using System;
using System.Collections.Generic;

namespace IncidentCommander
{
    /// <summary>Raised when a caller asks for a move the machine forbids.</summary>
    public sealed class IllegalTransitionException : Exception
    {
        /// <summary>Create the exception for a rejected <paramref name="from"/> to
        /// <paramref name="to"/> move.</summary>
        public IllegalTransitionException(string from, string to)
            : base($"illegal transition {from} -> {to}")
        {
            From = from;
            To = to;
        }

        /// <summary>The state the move started from.</summary>
        public string From { get; }

        /// <summary>The rejected destination state.</summary>
        public string To { get; }
    }

    /// <summary>The deterministic incident lifecycle.</summary>
    public static class StateMachine
    {
        // Allowed maps each state to the states it may legally move to. A transition not
        // in this table is rejected - there is no path the machine will take that is not
        // declared here, so the flow cannot be steered somewhere unexpected by model output.
        private static readonly IReadOnlyDictionary<string, string[]> Allowed =
            new Dictionary<string, string[]>
            {
                [States.New] = new[] { States.Triaging },
                [States.Triaging] = new[] { States.Investigating, States.InvestigationFailed },
                [States.Investigating] =
                    new[] { States.RootCauseIdentified, States.InvestigationFailed },
                [States.RootCauseIdentified] =
                    new[] { States.RemediationProposed, States.Resolved, States.Escalated },
                [States.RemediationProposed] =
                    new[] { States.WaitingForApproval, States.Remediating, States.Escalated },
                [States.WaitingForApproval] = new[] { States.Remediating, States.Escalated },
                [States.Remediating] = new[] { States.Verifying, States.RemediationFailed },
                [States.Verifying] = new[] { States.Resolved, States.VerificationFailed },
                [States.Resolved] = Array.Empty<string>(),
                [States.InvestigationFailed] = Array.Empty<string>(),
                [States.RemediationFailed] = Array.Empty<string>(),
                [States.VerificationFailed] = Array.Empty<string>(),
                [States.Escalated] = Array.Empty<string>(),
            };

        private static readonly HashSet<string> TerminalStates = new HashSet<string>
        {
            States.Resolved,
            States.InvestigationFailed,
            States.RemediationFailed,
            States.VerificationFailed,
            States.Escalated,
        };

        /// <summary>Report whether <paramref name="from"/> to <paramref name="to"/> is a
        /// declared, legal move.</summary>
        public static bool CanTransition(string from, string to)
        {
            return Allowed.TryGetValue(from, out string[]? targets) && Array.IndexOf(targets, to) >= 0;
        }

        /// <summary>Report whether an incident in this state is finished.</summary>
        public static bool IsTerminal(string state)
        {
            return TerminalStates.Contains(state);
        }

        /// <summary>Create a fresh incident in the <c>NEW</c> state from an alert.</summary>
        public static Incident NewIncident(string id, Alert alert, DateTimeOffset now)
        {
            string title = $"{alert.Metric} on {alert.Service}";
            return new Incident
            {
                Id = id,
                Title = title,
                State = States.New,
                Alert = alert,
                CreatedAt = now,
                UpdatedAt = now,
                History = new List<Transition>
                {
                    new Transition
                    {
                        From = "",
                        To = States.New,
                        Reason = "incident opened",
                        At = now,
                    },
                },
            };
        }

        /// <summary>Advance the incident to a new state if the move is legal, recording it.
        /// Throws <see cref="IllegalTransitionException"/> and leaves the incident untouched
        /// otherwise - the machine never half-applies a move.</summary>
        public static void Transition(Incident inc, string to, string reason, DateTimeOffset now)
        {
            if (!CanTransition(inc.State, to))
            {
                throw new IllegalTransitionException(inc.State, to);
            }

            inc.History.Add(new Transition
            {
                From = inc.State,
                To = to,
                Reason = reason,
                At = now,
            });
            inc.State = to;
            inc.UpdatedAt = now;
        }
    }
}
