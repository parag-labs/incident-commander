//! A tiny, dependency-free counter/gauge registry that exposes Prometheus text-format
//! metrics.
//!
//! The only subtlety in porting this module is [`format_g`], which reproduces Go's `%g`
//! verb (shortest round-trippable digits, switching to scientific notation for exponents
//! below -4 or at/above 6) so that [`Metrics::expose`] is byte-identical to the Go
//! reference. A [`std::collections::BTreeMap`] gives the sorted iteration order the
//! reference achieves by sorting names on every call.

use std::collections::BTreeMap;

/// Format a float the way Go's `%g` verb does (shortest representation).
pub fn format_g(value: f64) -> String {
    if value == 0.0 {
        return "0".to_string();
    }
    let neg = value < 0.0;
    let a = if neg { -value } else { value };

    let mut digits = String::new();
    let mut exp = 0i32;
    for precision in 1..=17u32 {
        let candidate = format!("{:.*e}", (precision - 1) as usize, a);
        if candidate.parse::<f64>() == Ok(a) {
            let (mantissa, exp_str) = candidate.split_once('e').expect("scientific form has e");
            exp = exp_str.parse().expect("exponent is an integer");
            digits = mantissa.replace('.', "");
            break;
        }
    }
    if digits.is_empty() {
        let candidate = format!("{:.*e}", 16usize, a);
        let (mantissa, exp_str) = candidate.split_once('e').expect("scientific form has e");
        exp = exp_str.parse().expect("exponent is an integer");
        digits = mantissa.replace('.', "");
    }

    let trimmed = digits.trim_end_matches('0');
    let digits = if trimmed.is_empty() { "0" } else { trimmed };
    let out = emit(digits, exp);
    if neg {
        format!("-{out}")
    } else {
        out
    }
}

/// Assemble the final string from shortest digits and a decimal exponent, choosing fixed
/// or scientific notation exactly as Go's `%g` does.
fn emit(digits: &str, exp: i32) -> String {
    if (-4..6).contains(&exp) {
        if exp >= 0 {
            let int_len = (exp + 1) as usize;
            if digits.len() <= int_len {
                let mut s = String::from(digits);
                s.push_str(&"0".repeat(int_len - digits.len()));
                s
            } else {
                format!("{}.{}", &digits[..int_len], &digits[int_len..])
            }
        } else {
            format!("0.{}{}", "0".repeat((-exp - 1) as usize), digits)
        }
    } else {
        let mantissa = if digits.len() == 1 {
            digits.to_string()
        } else {
            format!("{}.{}", &digits[..1], &digits[1..])
        };
        let sign = if exp >= 0 { "+" } else { "-" };
        format!("{mantissa}e{sign}{:02}", exp.abs())
    }
}

/// A registry of named counters and gauges.
///
/// The Go reference guards its maps with a mutex for concurrent use; this port is a plain
/// value, since the deterministic behaviour - accumulation and sorted exposition - is
/// what matters for the decision core.
#[derive(Debug, Clone, Default)]
pub struct Metrics {
    counters: BTreeMap<String, f64>,
    gauges: BTreeMap<String, f64>,
}

impl Metrics {
    /// Build an empty registry.
    pub fn new() -> Self {
        Metrics::default()
    }

    /// Add `delta` to a counter.
    pub fn inc(&mut self, name: &str, delta: f64) {
        *self.counters.entry(name.to_string()).or_insert(0.0) += delta;
    }

    /// Record a gauge value.
    pub fn set(&mut self, name: &str, value: f64) {
        self.gauges.insert(name.to_string(), value);
    }

    /// Return the current value of a counter (0 if it does not exist).
    pub fn counter(&self, name: &str) -> f64 {
        *self.counters.get(name).unwrap_or(&0.0)
    }

    /// Render the registry in Prometheus text exposition format.
    pub fn expose(&self) -> String {
        let mut out = String::new();
        for (kind, vals) in [("counter", &self.counters), ("gauge", &self.gauges)] {
            for (name, value) in vals {
                out.push_str(&format!(
                    "# TYPE {name} {kind}\n{name} {}\n",
                    format_g(*value)
                ));
            }
        }
        out
    }
}

/// Build an empty registry (the reference's `NewMetrics`).
pub fn new_metrics() -> Metrics {
    Metrics::new()
}
