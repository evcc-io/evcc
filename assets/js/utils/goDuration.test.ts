import { describe, expect, test } from "vite-plus/test";
import parseGoDuration, { goDurationToUnit, goDurationUnit, toGoDuration } from "./goDuration";

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

describe("toGoDuration", () => {
  test("formats value with unit suffix", () => {
    expect(toGoDuration(6, "hour")).toBe("6h");
    expect(toGoDuration(90, "second")).toBe("90s");
    expect(toGoDuration(30, "minute")).toBe("30m");
    expect(toGoDuration(1.5, "hour")).toBe("1.5h");
    expect(toGoDuration(0.1, "second")).toBe("0.1s");
    expect(toGoDuration(0)).toBe("0s");
    expect(toGoDuration(5, "fortnight")).toBe("5s");
  });

  test("roundtrips through parseGoDuration", () => {
    expect(parseGoDuration(toGoDuration(6, "hour"))).toBe(6 * 3.6e12);
    expect(parseGoDuration(toGoDuration(1.5, "minute"))).toBe(90e9);
  });
});

describe("goDurationUnit", () => {
  test("detects single-unit strings", () => {
    expect(goDurationUnit("12h")).toBe("hour");
    expect(goDurationUnit("90s")).toBe("second");
    expect(goDurationUnit("30m")).toBe("minute");
    expect(goDurationUnit("1.5h")).toBe("hour");
  });

  test("null for composite, numbers, empty", () => {
    expect(goDurationUnit("1h30m")).toBeNull();
    expect(goDurationUnit("")).toBeNull();
    expect(goDurationUnit(15000000000)).toBeNull();
    expect(goDurationUnit("6")).toBeNull();
    expect(goDurationUnit("500ms")).toBeNull();
  });
});
