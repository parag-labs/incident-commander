// A tiny, dependency-free counter/gauge registry that exposes Prometheus text-format
// metrics.
//
// The only subtlety in porting this module is FormatG, which reproduces Go's `%g` verb
// (shortest round-trippable digits, switching to scientific notation for exponents below
// -4 or at/above 6) so that Expose is byte-identical to the Go reference. Ordinal-sorted
// maps give the deterministic order the reference achieves by sorting names on every call.

using System;
using System.Collections.Generic;
using System.Globalization;
using System.Text;

namespace IncidentCommander
{
    /// <summary>A registry of named counters and gauges.</summary>
    public sealed class Metrics
    {
        private readonly SortedDictionary<string, double> _counters =
            new SortedDictionary<string, double>(StringComparer.Ordinal);

        private readonly SortedDictionary<string, double> _gauges =
            new SortedDictionary<string, double>(StringComparer.Ordinal);

        /// <summary>Add <paramref name="delta"/> to a counter.</summary>
        public void Inc(string name, double delta)
        {
            _counters.TryGetValue(name, out double current);
            _counters[name] = current + delta;
        }

        /// <summary>Record a gauge value.</summary>
        public void Set(string name, double value)
        {
            _gauges[name] = value;
        }

        /// <summary>Return the current value of a counter (0 if it does not exist).</summary>
        public double Counter(string name)
        {
            _counters.TryGetValue(name, out double value);
            return value;
        }

        /// <summary>Render the registry in Prometheus text exposition format.</summary>
        public string Expose()
        {
            StringBuilder b = new StringBuilder();
            Write(b, "counter", _counters);
            Write(b, "gauge", _gauges);
            return b.ToString();
        }

        private static void Write(StringBuilder b, string kind, SortedDictionary<string, double> vals)
        {
            foreach (KeyValuePair<string, double> kv in vals)
            {
                b.Append("# TYPE ").Append(kv.Key).Append(' ').Append(kind).Append('\n');
                b.Append(kv.Key).Append(' ').Append(FormatG(kv.Value)).Append('\n');
            }
        }

        /// <summary>Format a float the way Go's <c>%g</c> verb does (shortest
        /// representation).</summary>
        public static string FormatG(double value)
        {
            if (value == 0.0)
            {
                return "0";
            }

            bool neg = value < 0.0;
            double a = neg ? -value : value;

            string digits = "";
            int exp = 0;
            for (int precision = 1; precision <= 17; precision++)
            {
                string format = "E" + (precision - 1).ToString(CultureInfo.InvariantCulture);
                string candidate = a.ToString(format, CultureInfo.InvariantCulture);
                if (double.Parse(candidate, CultureInfo.InvariantCulture) == a)
                {
                    (digits, exp) = SplitScientific(candidate);
                    break;
                }
            }

            if (digits.Length == 0)
            {
                string candidate = a.ToString("E16", CultureInfo.InvariantCulture);
                (digits, exp) = SplitScientific(candidate);
            }

            digits = digits.TrimEnd('0');
            if (digits.Length == 0)
            {
                digits = "0";
            }

            string emitted = Emit(digits, exp);
            return neg ? "-" + emitted : emitted;
        }

        private static (string Digits, int Exp) SplitScientific(string candidate)
        {
            int eIdx = candidate.IndexOf('E');
            string mantissa = candidate.Substring(0, eIdx);
            int exp = int.Parse(candidate.Substring(eIdx + 1), CultureInfo.InvariantCulture);
            return (mantissa.Replace(".", ""), exp);
        }

        private static string Emit(string digits, int exp)
        {
            if (exp >= -4 && exp < 6)
            {
                if (exp >= 0)
                {
                    int intLen = exp + 1;
                    if (digits.Length <= intLen)
                    {
                        return digits + new string('0', intLen - digits.Length);
                    }

                    return digits.Substring(0, intLen) + "." + digits.Substring(intLen);
                }

                return "0." + new string('0', -exp - 1) + digits;
            }

            string mantissa = digits.Length == 1
                ? digits
                : digits.Substring(0, 1) + "." + digits.Substring(1);
            string sign = exp >= 0 ? "+" : "-";
            return mantissa + "e" + sign + Math.Abs(exp).ToString("D2", CultureInfo.InvariantCulture);
        }
    }
}
