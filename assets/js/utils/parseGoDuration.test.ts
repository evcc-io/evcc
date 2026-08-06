import { describe, expect, test } from "vite-plus/test";
import parseGoDuration, { goDurationToUnit } from "./parseGoDuration";

describe("parseGoDuration", () => {
  test("parses single unit", () => {
    expect(parseGoDuration("24h")).toBe(24 * 3.6e12);
    expect(parseGoDuration("30m")).toBe(30 * 6e10);
    expect(parseGoDuration("45s")).toBe(45e9);
    expect(parseGoDuration("500ms")).toBe(5e8);
    expect(parseGoDuration("10us")).toBe(1e4);
    expect(parseGoDuration("10µs")).toBe(1e4);
    expect(parseGoDuration("100ns")).toBe(100);
  });

  test("parses compound durations", () => {
    expect(parseGoDuration("1h30m")).toBe(3.6e12 + 30 * 6e10);
    expect(parseGoDuration("1m30s")).toBe(6e10 + 30e9);
  });

  test("parses decimals", () => {
    expect(parseGoDuration("0.1s")).toBe(1e8);
    expect(parseGoDuration("1.5h")).toBe(1.5 * 3.6e12);
  });

  test("rejects invalid input", () => {
    expect(parseGoDuration("")).toBeNull();
    expect(parseGoDuration("6")).toBeNull();
    expect(parseGoDuration("1h30")).toBeNull();
    expect(parseGoDuration("abc")).toBeNull();
    expect(parseGoDuration("6 h")).toBeNull();
  });
});

describe("goDurationToUnit", () => {
  test("converts to display unit", () => {
    expect(goDurationToUnit("10s")).toBe(10);
    expect(goDurationToUnit("5m")).toBe(300);
    expect(goDurationToUnit("90s", "minute")).toBe(1.5);
    expect(goDurationToUnit("3h", "hour")).toBe(3);
    expect(goDurationToUnit("6", "hour")).toBeNull();
  });
});
