using System.Collections.Generic;
using Xunit;

namespace IncidentCommander.Tests
{
    public class ModelsTests
    {
        [Fact]
        public void AllServicesStableOrder()
        {
            Assert.Equal(new[] { "api", "payment", "database", "cache", "queue" }, Services.All);
        }

        [Fact]
        public void StateConstantsAreReadableStrings()
        {
            Assert.Equal("NEW", States.New);
            Assert.Equal("WAITING_FOR_APPROVAL", States.WaitingForApproval);
            Assert.Equal("low", Safety.Low);
            Assert.Equal("critical", Severity.Critical);
        }

        private static Hypothesis Hyp(string cause, double confidence)
        {
            return new Hypothesis { Cause = cause, Confidence = confidence };
        }

        [Fact]
        public void DiagnosisTopReturnsHighestConfidence()
        {
            Diagnosis d = new Diagnosis
            {
                Hypotheses = new List<Hypothesis> { Hyp("a", 0.3), Hyp("b", 0.9), Hyp("c", 0.5) },
            };
            Assert.Equal("b", d.Top()!.Cause);
        }

        [Fact]
        public void DiagnosisTopTiesKeepFirst()
        {
            Diagnosis d = new Diagnosis
            {
                Hypotheses = new List<Hypothesis> { Hyp("first", 0.8), Hyp("second", 0.8) },
            };
            Assert.Equal("first", d.Top()!.Cause);
        }

        [Fact]
        public void DiagnosisTopEmptyIsNull()
        {
            Assert.Null(new Diagnosis().Top());
        }

        [Fact]
        public void DiagnosisTopHandlesZeroConfidence()
        {
            Diagnosis d = new Diagnosis
            {
                Hypotheses = new List<Hypothesis> { Hyp("only", 0.0) },
            };
            Assert.Equal("only", d.Top()!.Cause);
        }
    }
}
