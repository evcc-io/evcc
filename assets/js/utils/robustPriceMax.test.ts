import { describe, expect, it } from "vitest";
import { robustPriceMax, PRICE_SPIKE_CLIP } from "./robustPriceMax";

describe("robustPriceMax", () => {
  it("caps at the threshold when a spike exceeds it", () => {
    // everyday 0.1..1.1 $/kWh with a 20 $/kWh spike
    const vals = [0.1, 0.3, 0.6, 1.1, 20];
    expect(robustPriceMax(vals, { threshold: 3 })).toBe(3);
  });

  it("does not clip legit peaks below the threshold", () => {
    // max 1.1 is a high-but-real peak, not a spike -> full range
    const vals = [0.1, 0.3, 0.6, 1.1];
    expect(robustPriceMax(vals, { threshold: 3 })).toBe(1.1);
  });

  it("returns the true max when no threshold is given", () => {
    expect(robustPriceMax([0.1, 0.5, 20])).toBe(20);
  });

  it("ignores non-finite values and handles empty input", () => {
    expect(robustPriceMax([])).toBe(0);
    expect(robustPriceMax([NaN, Infinity, 0.5, 20], { threshold: 3 })).toBe(3);
  });

  it("exposes a sensible default spike threshold", () => {
    expect(PRICE_SPIKE_CLIP).toBe(3);
  });
});
