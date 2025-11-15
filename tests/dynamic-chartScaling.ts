import { describe, it, expect } from "vitest";
import { calculateDynamicScale, normalizeToHeight } from "../assets/js/utils/chartScaling";

describe("calculateDynamicScale", () => {
  it("should calculate scale for normal price range", () => {
    const values = [0.1, 0.2, 0.3, 0.4];
    const result = calculateDynamicScale(values);

    expect(result.min).toBeLessThan(0.1);
    expect(result.max).toBeGreaterThan(0.4);
    expect(result.range).toBe(result.max - result.min);
  });

  it("should handle small variations with minimum margin", () => {
    const values = [0.25, 0.26, 0.27];
    const result = calculateDynamicScale(values);

    // Range is 0.02, 10% would be 0.002, but minMargin is 0.01
    expect(result.range).toBeGreaterThan(0.02);
    expect(result.min).toBeLessThan(0.25);
    expect(result.max).toBeGreaterThan(0.27);
  });

  it("should handle identical values", () => {
    const values = [0.3, 0.3, 0.3];
    const result = calculateDynamicScale(values);

    // With zero range, should use minMargin
    expect(result.min).toBeLessThan(0.3);
    expect(result.max).toBeGreaterThan(0.3);
    expect(result.range).toBeGreaterThan(0);
  });

  it("should filter out undefined values", () => {
    const values = [0.1, undefined, 0.3, undefined, 0.5];
    const result = calculateDynamicScale(values);

    expect(result.min).toBeLessThan(0.1);
    expect(result.max).toBeGreaterThan(0.5);
  });

  it("should handle negative values", () => {
    const values = [-0.1, 0.0, 0.1];
    const result = calculateDynamicScale(values);

    expect(result.min).toBeLessThan(-0.1);
    expect(result.max).toBeGreaterThan(0.1);
  });

  it("should return default scale for empty array", () => {
    const values: number[] = [];
    const result = calculateDynamicScale(values);

    expect(result.min).toBe(0);
    expect(result.max).toBe(1);
    expect(result.range).toBe(1);
  });

  it("should handle array with only undefined values", () => {
    const values = [undefined, undefined, undefined];
    const result = calculateDynamicScale(values);

    expect(result.min).toBe(0);
    expect(result.max).toBe(1);
    expect(result.range).toBe(1);
  });

  it("should respect custom margin percentage", () => {
    const values = [1.0, 2.0];
    const result5 = calculateDynamicScale(values, 5);
    const result20 = calculateDynamicScale(values, 20);

    // Higher margin should create larger range
    expect(result20.range).toBeGreaterThan(result5.range);
  });

  it("should handle NaN values", () => {
    const values = [0.1, NaN, 0.3];
    const result = calculateDynamicScale(values);

    expect(result.min).toBeLessThan(0.1);
    expect(result.max).toBeGreaterThan(0.3);
  });
});

describe("normalizeToHeight", () => {
  it("should normalize value to height percentage", () => {
    const min = 0;
    const range = 10;

    expect(normalizeToHeight(0, min, range)).toBe(10); // Minimum value -> 10%
    expect(normalizeToHeight(10, min, range)).toBe(90); // Maximum value -> 90%
    expect(normalizeToHeight(5, min, range)).toBe(50); // Middle value -> 50%
  });

  it("should handle undefined values with middle height", () => {
    const result = normalizeToHeight(undefined, 0, 10);
    expect(result).toBe(50); // Middle of 10% and 90%
  });

  it("should handle zero range with middle height", () => {
    const result = normalizeToHeight(5, 5, 0);
    expect(result).toBe(50);
  });

  it("should respect custom height boundaries", () => {
    const min = 0;
    const range = 10;

    expect(normalizeToHeight(0, min, range, 20, 80)).toBe(20);
    expect(normalizeToHeight(10, min, range, 20, 80)).toBe(80);
    expect(normalizeToHeight(5, min, range, 20, 80)).toBe(50);
  });

  it("should handle negative min values", () => {
    const min = -5;
    const range = 10; // -5 to 5

    expect(normalizeToHeight(-5, min, range)).toBe(10);
    expect(normalizeToHeight(5, min, range)).toBe(90);
    expect(normalizeToHeight(0, min, range)).toBe(50);
  });

  it("should handle NaN values with middle height", () => {
    const result = normalizeToHeight(NaN, 0, 10);
    expect(result).toBe(50);
  });

  it("should clamp values within range", () => {
    const min = 0;
    const range = 10;

    // Values outside range should still be normalized
    const below = normalizeToHeight(-5, min, range);
    const above = normalizeToHeight(15, min, range);

    expect(below).toBeLessThan(10);
    expect(above).toBeGreaterThan(90);
  });
});
