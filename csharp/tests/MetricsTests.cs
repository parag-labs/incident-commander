using Xunit;

namespace IncidentCommander.Tests
{
    public class MetricsTests
    {
        [Fact]
        public void CounterAccumulates()
        {
            Metrics m = new Metrics();
            m.Inc("incident_count", 1);
            m.Inc("incident_count", 2);
            Assert.Equal(3.0, m.Counter("incident_count"));
        }

        [Fact]
        public void CounterMissingIsZero()
        {
            Assert.Equal(0.0, new Metrics().Counter("nope"));
        }

        [Fact]
        public void ExposeContainsCounterGaugeAndTypeLines()
        {
            Metrics m = new Metrics();
            m.Inc("incident_count", 1);
            m.Inc("incident_count", 2);
            m.Set("db_connections", 42.0);
            string outp = m.Expose();
            Assert.Contains("incident_count 3", outp);
            Assert.Contains("db_connections 42", outp);
            Assert.Contains("# TYPE incident_count counter", outp);
            Assert.Contains("# TYPE db_connections gauge", outp);
        }

        [Fact]
        public void ExposeIsSortedByName()
        {
            Metrics m = new Metrics();
            m.Inc("b_metric", 1);
            m.Inc("a_metric", 1);
            string outp = m.Expose();
            Assert.True(outp.IndexOf("a_metric", System.StringComparison.Ordinal)
                < outp.IndexOf("b_metric", System.StringComparison.Ordinal));
        }

        [Fact]
        public void ExposeExactLayout()
        {
            Metrics m = new Metrics();
            m.Inc("z_counter", 3);
            m.Inc("a_counter", 1);
            m.Set("b_gauge", 1.5);
            m.Set("a_gauge", 0.2);
            string expected =
                "# TYPE a_counter counter\n" +
                "a_counter 1\n" +
                "# TYPE z_counter counter\n" +
                "z_counter 3\n" +
                "# TYPE a_gauge gauge\n" +
                "a_gauge 0.2\n" +
                "# TYPE b_gauge gauge\n" +
                "b_gauge 1.5\n";
            Assert.Equal(expected, m.Expose());
        }

        [Fact]
        public void ExposeEmptyIsBlank()
        {
            Assert.Equal("", new Metrics().Expose());
        }

        [Fact]
        public void SetOverwritesGauge()
        {
            Metrics m = new Metrics();
            m.Set("queue_depth", 3.0);
            m.Set("queue_depth", 1.0);
            Assert.Contains("queue_depth 1\n", m.Expose());
        }

        [Fact]
        public void FormatGIntegers()
        {
            Assert.Equal("0", Metrics.FormatG(0.0));
            Assert.Equal("1", Metrics.FormatG(1.0));
            Assert.Equal("3", Metrics.FormatG(3.0));
            Assert.Equal("42", Metrics.FormatG(42.0));
            Assert.Equal("1000", Metrics.FormatG(1000.0));
        }

        [Fact]
        public void FormatGSimpleDecimals()
        {
            Assert.Equal("0.2", Metrics.FormatG(0.2));
            Assert.Equal("1.5", Metrics.FormatG(1.5));
            Assert.Equal("0.0001", Metrics.FormatG(0.0001));
            Assert.Equal("0.25", Metrics.FormatG(0.25));
        }

        [Fact]
        public void FormatGNegative()
        {
            Assert.Equal("-1", Metrics.FormatG(-1.0));
            Assert.Equal("-0.5", Metrics.FormatG(-0.5));
        }
    }
}
