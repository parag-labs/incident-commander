// A tiny, dependency-free counter/gauge registry that exposes Prometheus text-format
// metrics, ported from the Go reference.
//
// The only subtlety in porting this module is formatG, which reproduces Go's `%g` verb
// (shortest round-trippable digits, switching to scientific notation for exponents below
// -4 or at/above 6) so that expose() is byte-identical to the Go reference. Keys are sorted
// on every call to match the reference's deterministic name ordering.

/** Format a float the way Go's `%g` verb does (shortest representation). */
export function formatG(value: number): string {
  if (value === 0) {
    return "0";
  }
  const neg = value < 0;
  const a = neg ? -value : value;

  let digits = "";
  let exp = 0;
  for (let precision = 1; precision <= 17; precision++) {
    const candidate = a.toExponential(precision - 1);
    if (Number(candidate) === a) {
      const eIdx = candidate.indexOf("e");
      exp = Number(candidate.slice(eIdx + 1));
      digits = candidate.slice(0, eIdx).replace(".", "");
      break;
    }
  }
  if (digits === "") {
    const candidate = a.toExponential(16);
    const eIdx = candidate.indexOf("e");
    exp = Number(candidate.slice(eIdx + 1));
    digits = candidate.slice(0, eIdx).replace(".", "");
  }

  let end = digits.length;
  while (end > 1 && digits[end - 1] === "0") {
    end--;
  }
  digits = digits.slice(0, end);

  const out = emit(digits, exp);
  return neg ? `-${out}` : out;
}

function emit(digits: string, exp: number): string {
  if (exp >= -4 && exp < 6) {
    if (exp >= 0) {
      const intLen = exp + 1;
      if (digits.length <= intLen) {
        return digits + "0".repeat(intLen - digits.length);
      }
      return `${digits.slice(0, intLen)}.${digits.slice(intLen)}`;
    }
    return `0.${"0".repeat(-exp - 1)}${digits}`;
  }
  const mantissa = digits.length === 1 ? digits : `${digits[0]}.${digits.slice(1)}`;
  const sign = exp >= 0 ? "+" : "-";
  return `${mantissa}e${sign}${String(Math.abs(exp)).padStart(2, "0")}`;
}

/** A registry of named counters and gauges. */
export class Metrics {
  private readonly counters = new Map<string, number>();
  private readonly gauges = new Map<string, number>();

  /** Add `delta` to a counter. */
  inc(name: string, delta: number): void {
    this.counters.set(name, (this.counters.get(name) ?? 0) + delta);
  }

  /** Record a gauge value. */
  set(name: string, value: number): void {
    this.gauges.set(name, value);
  }

  /** Return the current value of a counter (0 if it does not exist). */
  counter(name: string): number {
    return this.counters.get(name) ?? 0;
  }

  /** Render the registry in Prometheus text exposition format. */
  expose(): string {
    return this.write("counter", this.counters) + this.write("gauge", this.gauges);
  }

  private write(kind: string, vals: Map<string, number>): string {
    let out = "";
    for (const name of [...vals.keys()].sort()) {
      out += `# TYPE ${name} ${kind}\n${name} ${formatG(vals.get(name) as number)}\n`;
    }
    return out;
  }
}

/** Build an empty registry (the reference's `NewMetrics`). */
export function newMetrics(): Metrics {
  return new Metrics();
}
