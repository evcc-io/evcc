import { describe, expect, test } from "vite-plus/test";
import { POWER_UNIT } from "../mixins/formatter";
import { axisScale, energyAxisScale, niceCeil } from "./energyAxis";

describe("niceCeil", () => {
  test("rounds up to nice numbers", () => {
    expect(niceCeil(0)).toBe(0);
    expect(niceCeil(60)).toBe(60);
    expect(niceCeil(5)).toBe(6);
    expect(niceCeil(700)).toBe(800);
    expect(niceCeil(1000)).toBe(1000);
    expect(niceCeil(15500)).toBe(20000);
  });
});

describe("axisScale", () => {
  test("steps between base and kilo unit at 1000", () => {
    expect(axisScale(402)).toEqual({ small: true, digits: 0, limit: 1000 });
    expect(axisScale(2500)).toEqual({ small: false, digits: 1, limit: 3000 });
  });
});

describe("energyAxisScale", () => {
  test("switches to W below 1 kW and floors the axis at 1000", () => {
    expect(energyAxisScale(600)).toEqual({ unit: POWER_UNIT.W, digits: 0, limit: 1000 });
  });

  test("zero data uses W scale", () => {
    expect(energyAxisScale(0)).toEqual({ unit: POWER_UNIT.W, digits: 0, limit: 1000 });
  });

  test("1-3 kW band gets one decimal to avoid duplicate integer ticks", () => {
    expect(energyAxisScale(1000)).toEqual({ unit: POWER_UNIT.KW, digits: 1, limit: 1000 });
    expect(energyAxisScale(2500)).toEqual({ unit: POWER_UNIT.KW, digits: 1, limit: 3000 });
  });

  test("larger peaks use kW without decimals", () => {
    expect(energyAxisScale(3500)).toEqual({ unit: POWER_UNIT.KW, digits: 0, limit: 4000 });
    expect(energyAxisScale(15500)).toEqual({ unit: POWER_UNIT.KW, digits: 0, limit: 20000 });
  });
});
