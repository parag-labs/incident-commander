"""Tests for metric aggregation and the Go %g compatible formatter.

Mirrors the reference obs/metrics_test.go (TestCounterAndGauge, TestExposeIsStable)
and pins format_g on version-independent values.
"""

from __future__ import annotations

import metrics as mx


def test_counter_accumulates() -> None:
    m = mx.new_metrics()
    m.inc("incident_count", 1)
    m.inc("incident_count", 2)
    assert m.counter("incident_count") == 3.0


def test_counter_missing_is_zero() -> None:
    assert mx.new_metrics().counter("nope") == 0.0


def test_expose_contains_counter_gauge_and_type_lines() -> None:
    m = mx.new_metrics()
    m.inc("incident_count", 1)
    m.inc("incident_count", 2)
    m.set("db_connections", 42.0)
    out = m.expose()
    assert "incident_count 3" in out
    assert "db_connections 42" in out
    assert "# TYPE incident_count counter" in out
    assert "# TYPE db_connections gauge" in out


def test_expose_is_sorted_by_name() -> None:
    m = mx.new_metrics()
    m.inc("b_metric", 1)
    m.inc("a_metric", 1)
    out = m.expose()
    assert out.index("a_metric") < out.index("b_metric")


def test_expose_exact_layout() -> None:
    m = mx.new_metrics()
    m.inc("z_counter", 3)
    m.inc("a_counter", 1)
    m.set("b_gauge", 1.5)
    m.set("a_gauge", 0.2)
    expected = (
        "# TYPE a_counter counter\n"
        "a_counter 1\n"
        "# TYPE z_counter counter\n"
        "z_counter 3\n"
        "# TYPE a_gauge gauge\n"
        "a_gauge 0.2\n"
        "# TYPE b_gauge gauge\n"
        "b_gauge 1.5\n"
    )
    assert m.expose() == expected


def test_expose_empty_is_blank() -> None:
    assert mx.new_metrics().expose() == ""


def test_set_overwrites_gauge() -> None:
    m = mx.new_metrics()
    m.set("queue_depth", 3.0)
    m.set("queue_depth", 1.0)
    assert "queue_depth 1\n" in m.expose()


def test_format_g_integers() -> None:
    assert mx.format_g(0.0) == "0"
    assert mx.format_g(1.0) == "1"
    assert mx.format_g(3.0) == "3"
    assert mx.format_g(42.0) == "42"
    assert mx.format_g(1000.0) == "1000"


def test_format_g_simple_decimals() -> None:
    assert mx.format_g(0.2) == "0.2"
    assert mx.format_g(1.5) == "1.5"
    assert mx.format_g(0.0001) == "0.0001"
    assert mx.format_g(0.25) == "0.25"


def test_format_g_negative() -> None:
    assert mx.format_g(-1.0) == "-1"
    assert mx.format_g(-0.5) == "-0.5"
