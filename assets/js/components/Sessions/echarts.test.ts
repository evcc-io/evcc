import { describe, expect, test } from "vite-plus/test";
import { sqrtTicks } from "./echarts";

describe("sqrtTicks", () => {
  test("percentage range", () => {
    expect(sqrtTicks(100)).toEqual([25, 50, 100]);
  });
  test("rounds up to next top", () => {
    expect(sqrtTicks(180)).toEqual([50, 100, 200]);
    expect(sqrtTicks(0.31)).toEqual([0.1, 0.2, 0.4]);
    expect(sqrtTicks(650)).toEqual([200, 400, 800]);
  });
  test("empty data", () => {
    expect(sqrtTicks(0)).toEqual([0.25, 0.5, 1]);
  });
});
