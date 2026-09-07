# Design

This document explains why the AI Incident Commander is built the way it is, the
trade-offs taken on purpose, and what it deliberately does **not** try to be.

## The one rule everything follows

> The LLM proposes and reasons. Deterministic code validates and executes.

Every structural decision below is downstream of that. The model is powerful but
untrustworthy for *action*: it can hallucinate a tool, misattribute a root cause, or
recommend something dangerous with high confidence. So the model is confined to a single
job — turn evidence into a structured hypothesis — and everything it produces is checked
by deterministic Go before it can affect the world.

## The pipeline, and where trust lives

```
alert → triage → investigate → diagnose → validate → risk-gate → execute → verify → report
        └─────── deterministic ───────┘   └ model ┘   └──────── deterministic ────────┘
```

- **Investigation** is deterministic and concurrent. A worker pool fans evidence
  collection across every service, respects context cancellation, and aggregates results
  in a stable order so evidence IDs are identical run to run despite the parallelism.
- **Diagnosis** is the only model step. It receives the evidence bundle and returns raw
  JSON. It never sees the environment and never calls a tool.
- **Validation** is the trust boundary. The raw response is parsed and hard-checked:
  is the recommended action a real tool? does every cited evidence ID exist? is the
  confidence in range, the service real, the risk a known level? Any violation is a typed
  error and the incident **fails safe to a human** — it never acts on output it couldn't
  verify.
- **The risk gate** is deterministic policy. It classifies the (already-validated) action
  by the tool's declared safety level and decides whether it may run automatically.
- **Execution and verification** are deterministic. The tool runs against the simulator,
  then the verifier compares before/after metrics to decide if it actually helped.

## Decisions taken on purpose

### A hand-written state machine, not model-driven control flow

The incident lifecycle is a table of legal transitions. `Transition` refuses any move
not in the table and never half-applies one. This is what makes an AI-in-the-loop system
auditable: no matter what the model says, the incident can only ever be in a declared
state, reached by a declared move. The model has no handle on it.

### The model returns text; the app owns parsing

`LLMClient.Diagnose` returns a raw string, not a parsed struct. That looks like extra
work, but it is the point: parsing and validation are *ours*, in deterministic code we
test directly, so a malformed or adversarial response is caught in one place. The AI
tests drive this boundary with exactly the failures a real model produces — malformed
JSON, a hallucinated tool, an unknown evidence ID, an out-of-range confidence.

### A deterministic mock reasoner instead of a recorded transcript

The mock model isn't a canned lookup table; it's a small rule-based root-cause analyzer
that groups evidence by service, scores each unhealthy service by how far its metrics
deviate from baseline, picks the worst as the root, and maps the dominant symptom to a
remediation. It only ever cites evidence it was given, so it cannot hallucinate. This
makes the whole system runnable and testable — and the evaluation meaningful — with no
API key, while the exact same agent code runs against a live model in production.

### Three safety levels, enforced in the tool layer

LOW / MEDIUM / HIGH aren't a suggestion in a prompt; they're a property of each tool that
the executor enforces. LOW runs automatically, MEDIUM runs only under auto-approval, and
HIGH can *only* run through an explicit human-approval path — there is no capability
setting that makes a rollback automatic. The evaluation asserts zero unsafe executions.

### Standard library over frameworks

`net/http` with Go 1.22 method routing, `log/slog`, and a ~40-line Prometheus-text metrics
registry. The spec called for Prometheus-compatible metrics and structured logging; both
are achievable without dependencies, and fewer dependencies is fewer things to break.

### In-memory storage behind an interface

Persistence is a stretch goal, not a prerequisite for proving the core. The `Store`
interface means a SQLite or Postgres implementation drops in without touching the agent,
API, or state machine.

## Non-goals

- **Not a production incident platform.** The environment is a deterministic simulator,
  not real infrastructure — by design, so behaviour is provable and nothing dangerous can
  happen. Connecting adapters to real telemetry/actuators is out of scope for v1.
- **No persistence, multi-tenancy, or auth.** In-memory store, single tenant, no login.
- **No streaming, no distributed deployment.** One process; a bounded, synchronous
  investigation per incident.
- **No claim that the mock "is" an LLM.** It's a deterministic stand-in that exercises the
  same interfaces and guardrails. Root-cause accuracy of 100% reflects the fixed scenarios
  and the rule-based reasoner; against a live model the harness stays the same but the
  numbers become a real quality signal.
- **Not a general agent framework.** It does one job — incident response over a small,
  known service graph — and does it end to end, rather than being a toolkit for building
  arbitrary agents.

The goal is a small, honest, fully-tested demonstration of the one thing that actually
matters when you put a model near production: a hard, deterministic boundary between what
the model *says* and what the system *does*.
