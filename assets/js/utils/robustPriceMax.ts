// Upper bound for price charts.
//
// Dynamic tariffs (e.g. AU AEMO/Amber) occasionally spike from a normal range
// into the dollars per kWh. Auto-scaling the axis to that peak flattens the
// everyday range into an unreadable line.
//
// A fixed spike threshold caps the axis: prices above it clip at the ceiling
// while everything below keeps a natural linear scale. Unlike a percentile cap
// this never clips legitimately-high-but-not-spike peaks — only prices that
// exceed the threshold are treated as spikes.
//
// The threshold is expressed in the same unit as the values passed in; callers
// scale PRICE_SPIKE_CLIP (main currency unit per kWh) to their chart's unit.

export const PRICE_SPIKE_CLIP = 3; // clip prices above 3 /kWh (e.g. AU $3/kWh)

export interface RobustPriceMaxOptions {
  threshold?: number; // clip above this (chart units); no clipping when unset
}

/**
 * Returns the chart's upper bound: the threshold when a spike exceeds it,
 * otherwise the true maximum (so non-spike data scales to its natural range).
 */
export function robustPriceMax(values: number[], opts: RobustPriceMaxOptions = {}): number {
  const { threshold } = opts;
  const finite = values.filter((v) => Number.isFinite(v));
  if (finite.length === 0) return 0;

  const trueMax = Math.max(...finite);
  return threshold != null && threshold > 0 && trueMax > threshold ? threshold : trueMax;
}
