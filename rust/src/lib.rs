//! Deterministic decision core of the AI Incident Commander, ported from Go.
//!
//! This crate contains only the pure, dependency-free algorithmic core: the domain
//! types, the incident state machine, the risk-gating policy, the evaluation scoring,
//! and the metrics registry. The LLM agent, HTTP API, storage, tool execution, and
//! environment simulator deliberately remain in the Go reference - they are not
//! deterministic and are out of scope for a portable core.

pub mod evaluation;
pub mod metrics;
pub mod models;
pub mod policy;
pub mod statemachine;
