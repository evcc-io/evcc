import { describe, expect, test } from "vite-plus/test";
import { slotStepData } from "./echarts";

const QUARTER = 15 * 60 * 1000;
const at = (h: number, m = 0) => new Date(2025, 0, 1, h, m).getTime();
const slot = (start: number, value: number) => ({ start, end: start + QUARTER, value });

describe("slotStepData", () => {
  const start = new Date(at(23));
  const end = new Date(at(24));

  test("starts each value at its slot start and closes the last slot", () => {
    const slots = [slot(at(23, 30), 30), slot(at(23, 45), 30), slot(at(24), 10)];
    expect(slotStepData(slots, start, new Date(at(24, 15)))).toEqual([
      [at(23, 30), 30, at(23, 30)],
      [at(23, 45), 30, at(23, 45)],
      [at(24), 10, at(24)],
      [at(24, 15), 10, at(24)],
    ]);
  });

  test("clamps to the chart range", () => {
    const hourly = { start: at(22, 30), end: at(23, 30), value: 20 };
    expect(slotStepData([hourly, slot(at(23, 30), 30)], start, new Date(at(23, 40)))).toEqual([
      [at(23), 20, at(23)],
      [at(23, 30), 30, at(23, 30)],
      [at(23, 40), 30, at(23, 30)],
    ]);
  });

  test("handles no slots", () => {
    expect(slotStepData([], start, end)).toEqual([]);
  });
});
