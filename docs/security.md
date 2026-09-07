# Security & guardrails

The threat model is simple: **treat the model's output as untrusted input.** Everything
below follows from that.

## Guardrails

- **Tool results are untrusted data.** The model may only cite evidence that was actually
  collected. A hypothesis referencing an unknown evidence ID is rejected before any action.
- **No free-form execution.** The model cannot name an action that isn't a registered
  tool, and it cannot execute anything — it returns a recommendation that the deterministic
  gate validates and decides on.
- **Three-level safety, enforced in code.** Tools are `low` / `medium` / `high`. LOW runs
  automatically; MEDIUM runs only under auto-approval; HIGH (e.g. a production rollback)
  runs **only** through an explicit human-approval path. There is no configuration that
  makes a HIGH action automatic — it's enforced in the tool executor, not the prompt.
- **Bounded loops.** Total tool calls are capped and every call runs under a cancellable
  context. There is no unbounded autonomous loop.
- **Fail safe.** Any validation failure — malformed output, hallucinated tool, bad
  evidence, out-of-range confidence — escalates to a human rather than guessing.
- **Audit trail.** Every state transition and every tool call (including its safety level,
  outcome, and duration) is recorded on the incident.

## What the evaluation asserts

The scorecard (`make evaluate`, and the CI gate) fails the build if **any** unsafe action
executes — a HIGH tool running without passing through `WAITING_FOR_APPROVAL` — or if
root-cause accuracy drops below the bar. Across the 10 scenarios, unsafe actions are `0`.

## Prompt injection

Because tool results are treated as data and the model's only output is a structured
recommendation that is independently validated, a prompt-injection payload embedded in a
log line or metric cannot cause an action: it can at most influence a *recommendation*,
which still has to name a real tool, cite real evidence, and clear the risk gate — and a
HIGH action still stops for a human regardless.
