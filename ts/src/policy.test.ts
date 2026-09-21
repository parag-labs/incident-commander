import { describe, expect, it } from "vitest";
import { Remediation, Safety } from "./models.js";
import { assess, type Config } from "./policy.js";

function rem(action: string): Remediation {
  return new Remediation({ action });
}

const disabled: Config = { autoApproveMedium: false };
const enabled: Config = { autoApproveMedium: true };

describe("policy", () => {
  it("always auto-approves low-risk actions", () => {
    const ra = assess(rem("query_metrics"), Safety.Low, disabled);
    expect(ra.autoApprovable).toBe(true);
    expect(ra.needsApproval).toBe(false);
    expect(ra.safety).toBe(Safety.Low);
    expect(ra.reason).toBe("read-only / low-risk action; safe to run automatically");
  });

  it("requires approval for medium when auto-approval is disabled", () => {
    const ra = assess(rem("restart_service"), Safety.Medium, disabled);
    expect(ra.autoApprovable).toBe(false);
    expect(ra.needsApproval).toBe(true);
    expect(ra.reason).toBe(
      "medium-risk action; requires human approval (auto-approval disabled)",
    );
  });

  it("auto-approves medium when enabled", () => {
    const ra = assess(rem("restart_service"), Safety.Medium, enabled);
    expect(ra.autoApprovable).toBe(true);
    expect(ra.needsApproval).toBe(false);
    expect(ra.reason).toBe("medium-risk action; auto-approval is enabled");
  });

  it("never auto-approves high-risk actions", () => {
    for (const cfg of [disabled, enabled]) {
      const ra = assess(rem("rollback_deployment"), Safety.High, cfg);
      expect(ra.autoApprovable).toBe(false);
      expect(ra.needsApproval).toBe(true);
      expect(ra.reason).toBe("high-risk action; never runs without explicit human approval");
    }
  });

  it("defaults unknown safety levels to human approval", () => {
    const ra = assess(rem("???"), "weird", enabled);
    expect(ra.autoApprovable).toBe(false);
    expect(ra.needsApproval).toBe(true);
    expect(ra.reason).toBe('unknown safety level "weird"; defaulting to human approval');
    expect(ra.safety).toBe("weird");
  });
});
