# Agent design

## The agent loop

The agent is a **bounded**, mostly-deterministic loop. It is not a free-running
autonomous agent; it runs one investigation to completion under hard limits.

```
collect evidence (concurrent) → diagnose (model) → validate → gate → act → verify
```

Limits (see `agent.Options`):

- `MaxToolCalls` caps total tool calls (default 40).
- One model call per investigation (the mock reasons in a single pass; a live model
  could be given a multi-step budget, still bounded).
- Every tool call runs under the request context, so the whole run is cancellable.

## What the model is asked for

The model receives an `EvidenceBundle` — the alert, the collected evidence (each item
carries an ID and structured detail), and recent deployments — and must return a single
JSON object:

```json
{
  "summary": "...",
  "hypotheses": [{ "cause": "...", "confidence": 0.0, "evidence_ids": ["e1"] }],
  "recommended_action": "reduce_db_connections",
  "recommended_service": "database",
  "risk": "low|medium|high",
  "needs_human_approval": true
}
```

The full instruction is in [`prompts/incident-investigator.md`](../prompts/incident-investigator.md).

## Validation (the trust boundary)

`agent.Validate` rejects, with a typed error:

- **malformed** output that isn't the expected JSON shape;
- a **hallucinated tool** — a `recommended_action` that isn't a real remediation tool;
- **unknown evidence** — a cited `evidence_id` that was never collected;
- an **out-of-range confidence** (outside `[0,1]`);
- an **invalid service** or an unknown risk level.

A rejected diagnosis fails safe: the incident is marked `INVESTIGATION_FAILED` and the
environment is never touched.

## The mock reasoner

For tests and the default demo, `llm.MockLLM` is a deterministic rule-based root-cause
analyzer: it groups evidence by service, scores each unhealthy service by deviation from
baseline, picks the worst as the root (which correctly finds the *upstream* cause in a
cascade), and maps the dominant symptom to a remediation. It cites only evidence it was
given, so it can't hallucinate — the same property the validator enforces on any model.
