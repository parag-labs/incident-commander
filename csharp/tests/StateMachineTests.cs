using System;
using Xunit;

namespace IncidentCommander.Tests
{
    public class StateMachineTests
    {
        private static readonly DateTimeOffset Epoch = DateTimeOffset.UnixEpoch;

        private static Incident NewTestIncident()
        {
            Alert alert = new Alert
            {
                Service = Services.Database,
                Metric = "error_rate",
                SeverityLevel = Severity.Critical,
            };
            return StateMachine.NewIncident("inc-1", alert, Epoch);
        }

        [Fact]
        public void NewStartsInNew()
        {
            Incident inc = NewTestIncident();
            Assert.Equal(States.New, inc.State);
            Assert.Single(inc.History);
            Assert.Equal(States.New, inc.History[0].To);
            Assert.Equal("", inc.History[0].From);
            Assert.Equal("incident opened", inc.History[0].Reason);
        }

        [Fact]
        public void NewSetsTitleFromAlert()
        {
            Assert.Equal("error_rate on database", NewTestIncident().Title);
        }

        [Fact]
        public void HappyPathWalksToResolved()
        {
            Incident inc = NewTestIncident();
            string[] path =
            {
                States.Triaging,
                States.Investigating,
                States.RootCauseIdentified,
                States.RemediationProposed,
                States.WaitingForApproval,
                States.Remediating,
                States.Verifying,
                States.Resolved,
            };
            foreach (string to in path)
            {
                StateMachine.Transition(inc, to, "step", Epoch);
            }

            Assert.Equal(States.Resolved, inc.State);
            Assert.Equal(9, inc.History.Count);
        }

        [Fact]
        public void IllegalTransitionIsRejectedAndDoesNotMutate()
        {
            Incident inc = NewTestIncident();
            IllegalTransitionException ex = Assert.Throws<IllegalTransitionException>(
                () => StateMachine.Transition(inc, States.Remediating, "skip", Epoch));
            Assert.Equal(States.New, ex.From);
            Assert.Equal(States.Remediating, ex.To);
            Assert.Equal(States.New, inc.State);
            Assert.Single(inc.History);
        }

        [Fact]
        public void IllegalTransitionMessage()
        {
            Incident inc = NewTestIncident();
            IllegalTransitionException ex = Assert.Throws<IllegalTransitionException>(
                () => StateMachine.Transition(inc, States.Resolved, "x", Epoch));
            Assert.Equal("illegal transition NEW -> RESOLVED", ex.Message);
        }

        [Fact]
        public void TerminalStatesHaveNoExit()
        {
            string[] terminals =
            {
                States.Resolved,
                States.Escalated,
                States.InvestigationFailed,
                States.RemediationFailed,
                States.VerificationFailed,
            };
            foreach (string state in terminals)
            {
                Assert.True(StateMachine.IsTerminal(state));
                Incident inc = NewTestIncident();
                inc.State = state;
                Assert.Throws<IllegalTransitionException>(
                    () => StateMachine.Transition(inc, States.Resolved, "x", Epoch));
            }
        }

        [Fact]
        public void NonTerminalStatesAreNotTerminal()
        {
            string[] states =
            {
                States.New,
                States.Triaging,
                States.Investigating,
                States.RootCauseIdentified,
                States.RemediationProposed,
                States.WaitingForApproval,
                States.Remediating,
                States.Verifying,
            };
            foreach (string state in states)
            {
                Assert.False(StateMachine.IsTerminal(state));
            }
        }

        [Fact]
        public void InvestigationCanFail()
        {
            Incident inc = NewTestIncident();
            StateMachine.Transition(inc, States.Triaging, "step", Epoch);
            StateMachine.Transition(inc, States.Investigating, "step", Epoch);
            StateMachine.Transition(inc, States.InvestigationFailed, "no evidence", Epoch);
            Assert.True(StateMachine.IsTerminal(inc.State));
        }

        [Fact]
        public void ReadOnlyResolutionSkipsRemediation()
        {
            Incident inc = NewTestIncident();
            StateMachine.Transition(inc, States.Triaging, "step", Epoch);
            StateMachine.Transition(inc, States.Investigating, "step", Epoch);
            StateMachine.Transition(inc, States.RootCauseIdentified, "step", Epoch);
            StateMachine.Transition(inc, States.Resolved, "informational", Epoch);
            Assert.Equal(States.Resolved, inc.State);
        }

        [Fact]
        public void CanTransitionMatchesTable()
        {
            Assert.True(StateMachine.CanTransition(States.New, States.Triaging));
            Assert.False(StateMachine.CanTransition(States.New, States.Resolved));
            Assert.True(StateMachine.CanTransition(States.RootCauseIdentified, States.Escalated));
            Assert.False(StateMachine.CanTransition(States.Resolved, States.Triaging));
        }

        [Fact]
        public void CanTransitionUnknownStateIsFalse()
        {
            Assert.False(StateMachine.CanTransition("BOGUS", States.Triaging));
        }

        [Fact]
        public void TransitionRecordsFromAndReason()
        {
            Incident inc = NewTestIncident();
            StateMachine.Transition(inc, States.Triaging, "begin triage", Epoch);
            Transition last = inc.History[inc.History.Count - 1];
            Assert.Equal(States.New, last.From);
            Assert.Equal(States.Triaging, last.To);
            Assert.Equal("begin triage", last.Reason);
            Assert.Equal(Epoch, inc.UpdatedAt);
        }
    }
}
