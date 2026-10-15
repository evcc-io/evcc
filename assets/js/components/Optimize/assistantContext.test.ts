import { describe, expect, test } from "vite-plus/test";
import { planContext, planHours } from "./assistantContext";
import { OptimizationStatus, type EvOpt } from "@/types/evcc";

// three quarter hour slots from 10:00 and one hourly slot from 11:00
const evopt: EvOpt = {
  updated: "2026-01-01T10:00:00Z",
  req: {
    batteries: [
      {
        s_min: 1000,
        s_max: 9000,
        s_initial: 5000,
        c_min: 0,
        c_max: 3000,
        d_max: 3000,
        p_a: 0.0002,
        charge_from_grid: true,
      },
    ],
    time_series: {
      dt: [900, 900, 900, 3600],
      gt: [100, 100, 100, 400],
      ft: [500, 500, 500, 0],
      p_N: [0.0002, 0.0002, 0.0004, 0.0003],
      p_E: [0.0001, 0.0001, 0.0001, 0.0001],
    },
  },
  res: {
    status: OptimizationStatus.OPTIMAL,
    objective_value: -1.5,
    batteries: [
      {
        charging_power: [400, 400, 400, 0],
        discharging_power: [0, 0, 0, 400],
        state_of_charge: [5400, 5800, 6200, 5800],
      },
    ],
    grid_import: [0, 0, 0, 0],
    grid_export: [0, 0, 0, 0],
    flow_direction: [0, 0, 0, 0],
  },
  details: {
    timestamp: [
      new Date(2026, 0, 1, 10, 0).toISOString(),
      new Date(2026, 0, 1, 10, 15).toISOString(),
      new Date(2026, 0, 1, 10, 30).toISOString(),
      new Date(2026, 0, 1, 11, 0).toISOString(),
    ],
    batteryDetails: [{ type: "battery", title: "Home", name: "bat", capacity: 10 }],
    demandDetails: null,
  },
};

describe("planHours", () => {
  const hours = planHours(evopt);

  test("folds quarter slots into their hour", () => {
    expect(hours).toHaveLength(2);
    expect(hours[0]!.pv).toBe(1500);
    expect(hours[0]!.batteries[0]!.charge).toBe(1200);
    expect(hours[0]!.batteries[0]!.soc).toBe(6200);
  });

  test("averages prices by duration", () => {
    expect(hours[0]!.price).toBeCloseTo((0.0002 + 0.0002 + 0.0004) / 3);
    expect(hours[1]!.price).toBeCloseTo(0.0003);
  });
});

describe("planContext", () => {
  const context = planContext(evopt, "EUR", "charge_before_export", "battery");

  test("names the documentation and the state key", () => {
    expect(context).toContain("https://github.com/evcc-io/optimizer");
    expect(context).toContain(".evopt.req");
  });

  test("lists battery parameters and hourly rows", () => {
    expect(context).toContain("0: Home (battery, 10 kWh): soc 5.0 kWh within 1.0..9.0 kWh");
    expect(context).toContain("may charge from grid, no grid discharge");
    expect(context).toContain(
      "2026-01-01 10:00 | 0.267 | 0.100 | 1.5 | 0.3 | 0.0/0.0 | 1.2/0.0/6.2"
    );
    expect(context).toContain(
      "2026-01-01 11:00 | 0.300 | 0.100 | 0.0 | 0.4 | 0.0/0.0 | 0.0/0.4/5.8"
    );
  });
});
