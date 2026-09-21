import { describe, expect, it } from "vitest";
import {
  ALL_SERVICES,
  Diagnosis,
  Hypothesis,
  Safety,
  Severities,
  States,
} from "./models.js";

describe("models", () => {
  it("lists all services in a stable order", () => {
    expect(ALL_SERVICES).toEqual(["api", "payment", "database", "cache", "queue"]);
  });

  it("exposes readable string constants", () => {
    expect(States.New).toBe("NEW");
    expect(States.WaitingForApproval).toBe("WAITING_FOR_APPROVAL");
    expect(Safety.Low).toBe("low");
    expect(Severities.Critical).toBe("critical");
  });

  it("returns the highest-confidence hypothesis from top()", () => {
    const d = new Diagnosis({
      hypotheses: [
        new Hypothesis({ cause: "a", confidence: 0.3 }),
        new Hypothesis({ cause: "b", confidence: 0.9 }),
        new Hypothesis({ cause: "c", confidence: 0.5 }),
      ],
    });
    expect(d.top()?.cause).toBe("b");
  });

  it("keeps the first hypothesis on ties", () => {
    const d = new Diagnosis({
      hypotheses: [
        new Hypothesis({ cause: "first", confidence: 0.8 }),
        new Hypothesis({ cause: "second", confidence: 0.8 }),
      ],
    });
    expect(d.top()?.cause).toBe("first");
  });

  it("returns undefined from top() when empty", () => {
    expect(new Diagnosis().top()).toBeUndefined();
  });

  it("handles a single zero-confidence hypothesis", () => {
    const d = new Diagnosis({ hypotheses: [new Hypothesis({ cause: "only", confidence: 0.0 })] });
    expect(d.top()?.cause).toBe("only");
  });
});
