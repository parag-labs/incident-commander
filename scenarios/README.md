# Scenarios

The **executable** scenario definitions live in Go, in
[`internal/simulator/scenarios.go`](../internal/simulator/scenarios.go) — that is the
single source of truth the tests, the demo, and `make evaluate` all use.

The YAML files here are human-readable descriptors of the same scenarios, handy for
skimming what each one injects and what the correct fix is. They are documentation, not
loaded at runtime. See [`docs/evaluation.md`](../docs/evaluation.md) for the full table
of all ten.
