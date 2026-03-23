import { describe, expect, test } from "vitest";
import { findLowestSumSlotIndex } from "./forecast";

describe("findLowestSumSlotIndex", () => {
  test("finds lowest sum with span of 4", () => {
    const slots = [
      { start: "2025-01-01T00:00:00Z", value: 10 },
      { start: "2025-01-01T00:15:00Z", value: 8 },
      { start: "2025-01-01T00:30:00Z", value: 6 },
      { start: "2025-01-01T00:45:00Z", value: 4 }, // sum 28 (index 0)
      { start: "2025-01-01T01:00:00Z", value: 2 }, // sum 20 (index 1)
      { start: "2025-01-01T01:15:00Z", value: 3 }, // sum 14 (index 2)
      { start: "2025-01-01T01:30:00Z", value: 5 }, // sum 13 (index 3) lowest
      { start: "2025-01-01T01:45:00Z", value: 7 }, // sum 17 (index 4)
    ];
    expect(findLowestSumSlotIndex(slots, 4)).toBe(3);
  });

  test("returns -1 when not enough slots", () => {
    const slots = [
      { start: "2025-01-01T00:00:00Z", value: 10 },
      { start: "2025-01-01T00:15:00Z", value: 8 },
    ];
    expect(findLowestSumSlotIndex(slots, 4)).toBe(-1);
  });

  test("handles exact span length", () => {
    const slots = [
      { start: "2025-01-01T00:00:00Z", value: 5 },
      { start: "2025-01-01T00:15:00Z", value: 3 },
      { start: "2025-01-01T00:30:00Z", value: 2 },
      { start: "2025-01-01T00:45:00Z", value: 1 },
    ];
    expect(findLowestSumSlotIndex(slots, 4)).toBe(0);
  });

  test("finds lowest at end", () => {
    const slots = [
      { start: "2025-01-01T00:00:00Z", value: 10 },
      { start: "2025-01-01T00:15:00Z", value: 10 },
      { start: "2025-01-01T00:30:00Z", value: 1 },
      { start: "2025-01-01T00:45:00Z", value: 1 },
      { start: "2025-01-01T01:00:00Z", value: 1 },
    ];
    expect(findLowestSumSlotIndex(slots, 3)).toBe(2);
  });

  test("returns first index when multiple equal sums", () => {
    const slots = [
      { start: "2025-01-01T00:00:00Z", value: 2 },
      { start: "2025-01-01T00:15:00Z", value: 2 }, // sum 4
      { start: "2025-01-01T00:30:00Z", value: 2 }, // sum 4 (same)
      { start: "2025-01-01T00:45:00Z", value: 2 },
    ];
    expect(findLowestSumSlotIndex(slots, 2)).toBe(0);
  });
});
