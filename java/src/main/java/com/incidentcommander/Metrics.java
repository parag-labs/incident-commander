package com.incidentcommander;

import java.util.Locale;
import java.util.Map;
import java.util.TreeMap;

/**
 * A tiny, dependency-free counter/gauge registry that exposes Prometheus text-format
 * metrics.
 *
 * <p>The only subtlety in porting this module is {@link #formatG(double)}, which reproduces
 * Go's {@code %g} verb (shortest round-trippable digits, switching to scientific notation
 * for exponents below -4 or at/above 6) so that {@link #expose()} is byte-identical to the
 * Go reference. {@link TreeMap} gives the deterministic name order the reference achieves
 * by sorting on every call.
 */
public final class Metrics {
    private final TreeMap<String, Double> counters = new TreeMap<>();
    private final TreeMap<String, Double> gauges = new TreeMap<>();

    /**
     * Add {@code delta} to a counter.
     *
     * @param name the counter name
     * @param delta the amount to add
     */
    public void inc(String name, double delta) {
        counters.merge(name, delta, Double::sum);
    }

    /**
     * Record a gauge value.
     *
     * @param name the gauge name
     * @param value the value to record
     */
    public void set(String name, double value) {
        gauges.put(name, value);
    }

    /**
     * Return the current value of a counter (0 if it does not exist).
     *
     * @param name the counter name
     * @return the counter value, or 0 when absent
     */
    public double counter(String name) {
        return counters.getOrDefault(name, 0.0);
    }

    /**
     * Render the registry in Prometheus text exposition format.
     *
     * @return the exposition text
     */
    public String expose() {
        StringBuilder b = new StringBuilder();
        write(b, "counter", counters);
        write(b, "gauge", gauges);
        return b.toString();
    }

    private static void write(StringBuilder b, String kind, TreeMap<String, Double> vals) {
        for (Map.Entry<String, Double> e : vals.entrySet()) {
            b.append("# TYPE ").append(e.getKey()).append(' ').append(kind).append('\n');
            b.append(e.getKey()).append(' ').append(formatG(e.getValue())).append('\n');
        }
    }

    /**
     * Build an empty registry (the reference's {@code NewMetrics}).
     *
     * @return a new, empty registry
     */
    public static Metrics newMetrics() {
        return new Metrics();
    }

    /**
     * Format a float the way Go's {@code %g} verb does (shortest representation).
     *
     * @param value the value to format
     * @return the shortest round-trippable decimal string
     */
    public static String formatG(double value) {
        if (value == 0.0) {
            return "0";
        }
        boolean neg = value < 0.0;
        double a = neg ? -value : value;

        String digits = "";
        int exp = 0;
        for (int precision = 1; precision <= 17; precision++) {
            String candidate = String.format(Locale.ROOT, "%." + (precision - 1) + "e", a);
            if (Double.parseDouble(candidate) == a) {
                int eIdx = candidate.indexOf('e');
                exp = Integer.parseInt(candidate.substring(eIdx + 1));
                digits = candidate.substring(0, eIdx).replace(".", "");
                break;
            }
        }
        if (digits.isEmpty()) {
            String candidate = String.format(Locale.ROOT, "%.16e", a);
            int eIdx = candidate.indexOf('e');
            exp = Integer.parseInt(candidate.substring(eIdx + 1));
            digits = candidate.substring(0, eIdx).replace(".", "");
        }

        int end = digits.length();
        while (end > 1 && digits.charAt(end - 1) == '0') {
            end--;
        }
        digits = digits.substring(0, end);

        String out = emit(digits, exp);
        return neg ? "-" + out : out;
    }

    private static String emit(String digits, int exp) {
        if (exp >= -4 && exp < 6) {
            if (exp >= 0) {
                int intLen = exp + 1;
                if (digits.length() <= intLen) {
                    StringBuilder sb = new StringBuilder(digits);
                    while (sb.length() < intLen) {
                        sb.append('0');
                    }
                    return sb.toString();
                }
                return digits.substring(0, intLen) + "." + digits.substring(intLen);
            }
            StringBuilder sb = new StringBuilder("0.");
            for (int i = 0; i < -exp - 1; i++) {
                sb.append('0');
            }
            return sb.append(digits).toString();
        }
        String mantissa = digits.length() == 1
                ? digits
                : digits.charAt(0) + "." + digits.substring(1);
        String sign = exp >= 0 ? "+" : "-";
        return mantissa + "e" + sign + String.format(Locale.ROOT, "%02d", Math.abs(exp));
    }
}
