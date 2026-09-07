# Architecture

The system is a single Go service around one rule: **the LLM proposes and reasons;
deterministic code validates and executes.**

## Components

| Package | Role | Deterministic? |
|---------|------|:--:|
| `internal/incident` | The incident state machine (legal transitions only) | ✅ |
| `internal/simulator` | Reproducible environment + the 10 scenarios | ✅ |
| `internal/tools` | Investigation + remediation tools (safety, authz, audit) | ✅ |
| `internal/policy` | The risk gate (low / medium / high) | ✅ |
| `internal/agent` | Orchestration, concurrent collection, output validation | ✅ (except the model call) |
| `internal/agent/llm` | Provider-agnostic model client (mock / scripted / OpenAI) | mock is ✅ |
| `internal/api` | `net/http` REST + read-only MCP | ✅ |
| `internal/eval` | The scenario scorecard | ✅ |
| `internal/obs` | Prometheus-text metrics | ✅ |
| `internal/storage` | In-memory incident store (swappable) | ✅ |
| `pkg/models` | Typed domain contracts | — |

## Request flow

1. `POST /incidents {"scenario": "..."}` builds a simulator environment and a `NEW` incident.
2. The orchestrator advances the state machine into `INVESTIGATING` and the collector fans
   read-only tools across every service with a worker pool.
3. The aggregated evidence bundle goes to the model, which returns a raw JSON diagnosis.
4. The diagnosis is validated (real tool? cited evidence exists? confidence in range?).
   On any failure the incident goes to `INVESTIGATION_FAILED` — it never acts on bad output.
5. The risk gate classifies the recommended action. LOW/auto-approved MEDIUM execute;
   HIGH (and MEDIUM without auto-approval) stop at `WAITING_FOR_APPROVAL`.
6. On execution, the tool runs against the simulator and the verifier compares before/after
   metrics; the incident resolves or fails verification.
7. `GET /incidents/{id}/report` returns the evidence-backed report plus a narrative.

## Concurrency

Evidence collection is the Go-concurrency core: a bounded worker pool pulls jobs from a
channel, each job runs a tool under the request context, and results are re-sorted by a
stable job index before evidence IDs are assigned. Cancelling the context aborts
in-flight collection. The result is parallel latency with deterministic output.
