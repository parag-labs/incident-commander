import { describe, expect, it } from "vitest";
import { formatG, newMetrics } from "./metrics.js";

describe("metrics registry", () => {
  it("accumulates a counter", () => {
    const m = newMetrics();
    m.inc("incident_count", 1);
    m.inc("incident_count", 2);
    expect(m.counter("incident_count")).toBe(3);
  });

  it("returns zero for a missing counter", () => {
    expect(newMetrics().counter("nope")).toBe(0);
  });

  it("exposes counters, gauges, and TYPE lines", () => {
    const m = newMetrics();
    m.inc("incident_count", 1);
    m.inc("incident_count", 2);
    m.set("db_connections", 42);
    const out = m.expose();
    expect(out).toContain("incident_count 3");
    expect(out).toContain("db_connections 42");
    expect(out).toContain("# TYPE incident_count counter");
    expect(out).toContain("# TYPE db_connections gauge");
  });

  it("exposes metrics sorted by name", () => {
    const m = newMetrics();
    m.inc("b_metric", 1);
    m.inc("a_metric", 1);
    const out = m.expose();
    expect(out.indexOf("a_metric")).toBeLessThan(out.indexOf("b_metric"));
  });

  it("produces an exact exposition layout", () => {
    const m = newMetrics();
    m.inc("z_counter", 3);
    m.inc("a_counter", 1);
    m.set("b_gauge", 1.5);
    m.set("a_gauge", 0.2);
    expect(m.expose()).toBe(
      "# TYPE a_counter counter\n" +
        "a_counter 1\n" +
        "# TYPE z_counter counter\n" +
        "z_counter 3\n" +
        "# TYPE a_gauge gauge\n" +
        "a_gauge 0.2\n" +
        "# TYPE b_gauge gauge\n" +
        "b_gauge 1.5\n",
    );
  });

  it("exposes an empty registry as the empty string", () => {
    expect(newMetrics().expose()).toBe("");
  });

  it("overwrites a gauge on set", () => {
    const m = newMetrics();
    m.set("queue_depth", 3);
    m.set("queue_depth", 1);
    expect(m.expose()).toContain("queue_depth 1\n");
  });
});

describe("formatG", () => {
  it("formats integers", () => {
    expect(formatG(0)).toBe("0");
    expect(formatG(1)).toBe("1");
    expect(formatG(3)).toBe("3");
    expect(formatG(42)).toBe("42");
    expect(formatG(1000)).toBe("1000");
  });

  it("formats simple decimals", () => {
    expect(formatG(0.2)).toBe("0.2");
    expect(formatG(1.5)).toBe("1.5");
    expect(formatG(0.0001)).toBe("0.0001");
    expect(formatG(0.25)).toBe("0.25");
  });

  it("formats negatives", () => {
    expect(formatG(-1)).toBe("-1");
    expect(formatG(-0.5)).toBe("-0.5");
  });
});
