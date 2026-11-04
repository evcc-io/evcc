import { describe, expect, test } from "vite-plus/test";
import { currentToPower, powerToCurrent } from "./phasePower";
import { GRID_CONNECTION } from "@/types/evcc";

describe("currentToPower", () => {
  test("single and three phase", () => {
    expect(currentToPower(10, 1)).toBe(2300);
    expect(currentToPower(16, 3)).toBe(11040);
  });

  test("three wire grid uses sqrt(3) on three phases", () => {
    expect(currentToPower(10, 3, 230, GRID_CONNECTION.THREE_PHASE_THREE_WIRE)).toBeCloseTo(
      3983.7,
      1
    );
    expect(currentToPower(10, 1, 230, GRID_CONNECTION.THREE_PHASE_THREE_WIRE)).toBe(2300);
  });
});

describe("powerToCurrent", () => {
  test("inverts currentToPower", () => {
    expect(powerToCurrent(2300, 1)).toBe(10);
    expect(powerToCurrent(11040, 3)).toBe(16);
    expect(powerToCurrent(500, 1)).toBeCloseTo(2.174, 3);
  });
});
