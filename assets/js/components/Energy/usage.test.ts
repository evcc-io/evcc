import { describe, expect, test } from "vite-plus/test";
import { usageSplit, type UsageSlot } from "./usage";

const slot = (s: Partial<UsageSlot>): UsageSlot => ({
  pv: 0,
  gridIn: 0,
  gridOut: 0,
  batOut: 0,
  home: 0,
  bats: [],
  lps: [],
  ...s,
});

describe("usageSplit", () => {
  test("production covers export, battery, home and loadpoints", () => {
    const [r] = usageSplit([slot({ pv: 2, gridOut: 0.5, bats: [0.5], home: 0.3, lps: [0.7] })]);
    expect(r).toEqual({ export: 0.5, bats: [0.5], home: 0.3, lps: [0.7] });
  });

  test("import and discharge feed what production left", () => {
    const [r] = usageSplit([
      slot({ pv: 1, gridIn: 1.5, batOut: 0.5, home: 1.25, bats: [0.25], lps: [1.5] }),
    ]);
    expect(r).toEqual({ export: 0, bats: [0.25], home: 1.25, lps: [1.5] });
  });

  test("hourly booked charger: import stays with consumption, the lump is capped", () => {
    const rows = usageSplit([
      slot({ gridIn: 3, home: 0.1, lps: [0] }),
      slot({ gridIn: 0.2, home: 0.1, lps: [6] }),
    ]);
    expect(rows[0]).toEqual({ export: 0, bats: [], home: 3, lps: [0] });
    expect(rows[1]!.home).toBeCloseTo(0.1);
    expect(rows[1]!.lps[0]).toBeCloseTo(0.1);
  });

  test("unclaimed energy stays with consumption", () => {
    const [r] = usageSplit([slot({ pv: 2, gridIn: 1, home: 0.5, lps: [0.5] })]);
    expect(r).toEqual({ export: 0, bats: [], home: 2.5, lps: [0.5] });
  });
});
