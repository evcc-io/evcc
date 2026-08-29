import { describe, expect, test } from "vite-plus/test";
import { expandForecast, findLowestSumSlotIndex, isStaticTariff } from "./forecast";

describe("expandForecast", () => {
  test("expands slots to milliseconds", () => {
    expect(expandForecast({ grid: [[1735689600, 1735693200, 0.25]] }).grid).toEqual([
      { start: 1735689600000, end: 1735693200000, value: 0.25 },
    ]);
  });

  test("expands the solar timeseries", () => {
    expect(expandForecast({ solar: { scale: 1, timeseries: [[1735689600, 1000]] } }).solar).toEqual(
      { scale: 1, timeseries: [{ ts: 1735689600000, val: 1000 }] }
    );
  });

  test("handles missing input", () => {
    expect(expandForecast(undefined)).toEqual({
      grid: undefined,
      co2: undefined,
      solar: undefined,
      planner: undefined,
      feedin: undefined,
      temperature: undefined,
    });
  });
});

describe("findLowestSumSlotIndex", () => {
  test("finds lowest sum with span of 4", () => {
    const slots = [
      { start: 1735689600000, value: 10 },
      { start: 1735690500000, value: 8 },
      { start: 1735691400000, value: 6 },
      { start: 1735692300000, value: 4 }, // sum 28 (index 0)
      { start: 1735693200000, value: 2 }, // sum 20 (index 1)
      { start: 1735694100000, value: 3 }, // sum 14 (index 2)
      { start: 1735695000000, value: 5 }, // sum 13 (index 3) lowest
      { start: 1735695900000, value: 7 }, // sum 17 (index 4)
    ];
    expect(findLowestSumSlotIndex(slots, 4)).toBe(3);
  });

  test("returns -1 when not enough slots", () => {
    const slots = [
      { start: 1735689600000, value: 10 },
      { start: 1735690500000, value: 8 },
    ];
    expect(findLowestSumSlotIndex(slots, 4)).toBe(-1);
  });

  test("handles exact span length", () => {
    const slots = [
      { start: 1735689600000, value: 5 },
      { start: 1735690500000, value: 3 },
      { start: 1735691400000, value: 2 },
      { start: 1735692300000, value: 1 },
    ];
    expect(findLowestSumSlotIndex(slots, 4)).toBe(0);
  });

  test("finds lowest at end", () => {
    const slots = [
      { start: 1735689600000, value: 10 },
      { start: 1735690500000, value: 10 },
      { start: 1735691400000, value: 1 },
      { start: 1735692300000, value: 1 },
      { start: 1735693200000, value: 1 },
    ];
    expect(findLowestSumSlotIndex(slots, 3)).toBe(2);
  });

  test("returns first index when multiple equal sums", () => {
    const slots = [
      { start: 1735689600000, value: 2 },
      { start: 1735690500000, value: 2 }, // sum 4
      { start: 1735691400000, value: 2 }, // sum 4 (same)
      { start: 1735692300000, value: 2 },
    ];
    expect(findLowestSumSlotIndex(slots, 2)).toBe(0);
  });
});

describe("isStaticTariff", () => {
  test("returns false for undefined", () => {
    expect(isStaticTariff(undefined)).toBe(false);
  });

  test("returns false for empty array", () => {
    expect(isStaticTariff([])).toBe(false);
  });

  test("returns true when all values are identical", () => {
    const slots = [
      { start: 1735689600000, end: 1735690500000, value: 0.25 },
      { start: 1735690500000, end: 1735691400000, value: 0.25 },
      { start: 1735691400000, end: 1735692300000, value: 0.25 },
    ];
    expect(isStaticTariff(slots)).toBe(true);
  });

  test("returns false when values differ", () => {
    const slots = [
      { start: 1735689600000, end: 1735690500000, value: 0.25 },
      { start: 1735690500000, end: 1735691400000, value: 0.3 },
      { start: 1735691400000, end: 1735692300000, value: 0.25 },
    ];
    expect(isStaticTariff(slots)).toBe(false);
  });
});
