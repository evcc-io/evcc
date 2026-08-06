const factors: Record<string, number> = {
  ns: 1,
  us: 1e3,
  µs: 1e3,
  ms: 1e6,
  s: 1e9,
  m: 6e10,
  h: 3.6e12,
};

export const displayFactors: Record<string, number> = { minute: 60, hour: 3600 };

// go duration string to display value in given Intl unit (default second), null if invalid
export function goDurationToUnit(value: string, unit?: string): number | null {
  const ns = parseGoDuration(value);
  return ns === null ? null : ns / (displayFactors[unit ?? ""] ?? 1) / 1e9;
}

// go duration string ("24h", "1h30m", "0.1s") to nanoseconds, null if invalid
export default function parseGoDuration(value: string): number | null {
  let total = 0;
  let matched = "";
  for (const [all, num, unit] of value.matchAll(/(\d+(?:\.\d+)?)(ns|us|µs|ms|s|m|h)/g)) {
    total += parseFloat(num!) * factors[unit!]!;
    matched += all;
  }
  return matched && matched === value ? total : null;
}
