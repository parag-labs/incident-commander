"""A tiny, dependency-free counter/gauge registry that exposes Prometheus text-format
metrics.

The only subtlety in porting this module is :func:`format_g`, which reproduces Go's
``%g`` verb (shortest round-trippable digits, switching to scientific notation for
exponents below -4 or at/above 6) so that :meth:`Metrics.expose` is byte-identical to
the Go reference.
"""

from __future__ import annotations


def format_g(value: float) -> str:
    """Format a float the way Go's ``%g`` verb does (shortest representation)."""
    if value == 0.0:
        return "0"
    neg = value < 0.0
    a = -value if neg else value

    digits = ""
    exp = 0
    for precision in range(1, 18):
        candidate = f"{a:.{precision - 1}e}"
        if float(candidate) == a:
            mantissa, _, exp_str = candidate.partition("e")
            exp = int(exp_str)
            digits = mantissa.replace(".", "")
            break
    else:  # pragma: no cover - float64 always round-trips within 17 digits
        candidate = f"{a:.16e}"
        mantissa, _, exp_str = candidate.partition("e")
        exp = int(exp_str)
        digits = mantissa.replace(".", "")

    digits = digits.rstrip("0") or "0"
    out = _emit(digits, exp)
    return "-" + out if neg else out


def _emit(digits: str, exp: int) -> str:
    if -4 <= exp < 6:
        if exp >= 0:
            int_len = exp + 1
            if len(digits) <= int_len:
                return digits + "0" * (int_len - len(digits))
            return digits[:int_len] + "." + digits[int_len:]
        return "0." + "0" * (-exp - 1) + digits
    mantissa = digits if len(digits) == 1 else digits[0] + "." + digits[1:]
    sign = "+" if exp >= 0 else "-"
    return f"{mantissa}e{sign}{abs(exp):02d}"


class Metrics:
    """A registry of named counters and gauges.

    The Go reference guards its maps with a mutex for concurrent use; this port is a
    plain object, since the deterministic behaviour - accumulation and sorted
    exposition - is what matters for the decision core.
    """

    def __init__(self) -> None:
        self._counters: dict[str, float] = {}
        self._gauges: dict[str, float] = {}

    def inc(self, name: str, delta: float) -> None:
        """Add ``delta`` to a counter."""
        self._counters[name] = self._counters.get(name, 0.0) + delta

    def set(self, name: str, value: float) -> None:
        """Record a gauge value."""
        self._gauges[name] = value

    def counter(self, name: str) -> float:
        """Return the current value of a counter (0 if it does not exist)."""
        return self._counters.get(name, 0.0)

    def expose(self) -> str:
        """Render the registry in Prometheus text exposition format."""
        parts: list[str] = []

        def write(kind: str, vals: dict[str, float]) -> None:
            for name in sorted(vals):
                parts.append(f"# TYPE {name} {kind}\n{name} {format_g(vals[name])}\n")

        write("counter", self._counters)
        write("gauge", self._gauges)
        return "".join(parts)


def new_metrics() -> Metrics:
    """Build an empty registry (the reference's ``NewMetrics``)."""
    return Metrics()
