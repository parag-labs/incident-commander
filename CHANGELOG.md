# Changelog

All notable changes to this project are documented here.

## [0.1.0] - 2026-09-06

First working version — the MVP end to end, fully tested with the deterministic model.

### Added
- Deterministic incident **state machine** (`NEW → … → RESOLVED` plus failure states)
  with legal-transition enforcement; the model never controls it.
- **Simulator** with a healthy baseline and 10 reproducible scenarios, including two
  where the alerting service is downstream of the true root (dependency timeout, cascade).
- **Investigation and remediation tools** with declared safety levels, typed I/O, context
  deadlines, authorization, and audit records.
- **Risk/safety gate** — LOW auto-runs, MEDIUM runs only under auto-approval, HIGH never
  runs without an explicit human approval.
- **Provider-agnostic LLM layer**: a deterministic mock reasoner, a scriptable stub for
  adversarial AI tests, and an OpenAI-compatible client (OpenAI / Azure / Ollama / vLLM).
- **Agent orchestration**: concurrent evidence collection via a worker pool, hard output
  validation (rejects malformed output, hallucinated tools, unknown evidence, bad
  confidence), and end-to-end remediation + verification.
- **HTTP API** (`net/http`): incident create/list/get, report, human approval, health,
  Prometheus-text metrics, and a read-only MCP endpoint.
- **Evaluation harness** (`make evaluate`) scoring all 10 scenarios; wired into CI as a
  gate. Current scorecard: 100% root-cause accuracy, 100% grounded, 0 unsafe actions.
- Docs (`architecture`, `agent-design`, `security`, `evaluation`), `DESIGN.md`, prompts,
  Dockerfile, docker-compose, Makefile, and a GitHub Actions CI pipeline (fmt, vet,
  `-race` tests, build, evaluate).
