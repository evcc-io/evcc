import { describe, expect, test } from "vitest";
import { calculateCostRange, findRateInRange, generateRateSlots } from "./tariffSlots";
import type { Rate, Slot } from "../types/evcc";

const d = new Date();
const now = new Date();
const h = 60 * 60 * 1000;
const t1 = new Date(now.getTime() + 1 * h);
const t1h30 = new Date(now.getTime() + 1.5 * h);
const t1h45 = new Date(now.getTime() + 1.75 * h);
const t2 = new Date(now.getTime() + 2 * h);
const t3 = new Date(now.getTime() + 3 * h);
const t4 = new Date(now.getTime() + 4 * h);

describe("calculateCostRange", () => {
  test("min/max", () => {
    const slots: Slot[] = [
      { day: "Mon", value: 10, start: d, end: d, charging: false },
      { day: "Mon", value: 5, start: d, end: d, charging: false },
      { day: "Mon", value: 15, start: d, end: d, charging: false },
    ];
    const { min, max } = calculateCostRange(slots);
    expect(min).toBe(5);
    expect(max).toBe(15);
  });

  test("undefined", () => {
    const slots: Slot[] = [
      { day: "Mon", value: 10, start: d, end: d, charging: false },
      { day: "Mon", value: undefined, start: d, end: d, charging: false },
    ];
    const { min, max } = calculateCostRange(slots);
    expect(min).toBe(10);
    expect(max).toBe(10);
  });
});

describe("findRateInRange", () => {
  test("overlap", () => {
    const r: Rate[] = [{ start: t1, end: t2, value: 0.15 }];
    expect(findRateInRange(t1h30, t1h45, r)?.value).toBe(0.15);
  });

  test("no match", () => {
    const r: Rate[] = [{ start: t3, end: t4, value: 0.25 }];
    expect(findRateInRange(t1, t1h45, r)).toBeUndefined();
  });
});

describe("generateRateSlots", () => {
  test("structure", () => {
    const r: Rate[] = [{ start: t1, end: t2, value: 0.15 }];
    const fmt = () => "Thu";
    const slots = generateRateSlots(r, fmt);
    expect(slots.length).toBeGreaterThan(0);
    expect(slots[0]).toHaveProperty("day");
    expect(slots[0]).toHaveProperty("value");
  });

  test("empty", () => {
    const fmt = () => "Thu";
    expect(generateRateSlots([], fmt)).toEqual([]);
  });

  test("formatter", () => {
    const r: Rate[] = [{ start: t1, end: t2, value: 0.15 }];
    const fmt = () => "X";
    const slots = generateRateSlots(r, fmt);
    expect(slots[0]?.day).toBe("X");
  });

  test("callbacks", () => {
    const r: Rate[] = [{ start: t1, end: t2, value: 0.15 }];
    const fmt = () => "Thu";
    const charging = (v: number | undefined) => v !== undefined && v < 0.2;
    const warning = (v: number | undefined) => v !== undefined && v > 0.2;
    const slots = generateRateSlots(r, fmt, charging, warning);
    const slot = slots.find((s) => s.value === 0.15);
    expect(slot?.charging).toBe(true);
    expect(slot?.warning).toBe(false);
  });
});
