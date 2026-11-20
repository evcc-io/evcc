import { describe, expect, test } from "vite-plus/test";
import { PRIORITY_BASIS, PRIORITY_STRATEGY } from "@/types/evcc";
import { changedPrioritySettings, type PriorityValues } from "./priority";

describe("changedPrioritySettings", () => {
  test("skips an invalid hidden hysteresis when disabling the strategy", () => {
    const serverValues: PriorityValues = {
      priorityStrategy: PRIORITY_STRATEGY.SOC,
      priorityBasis: PRIORITY_BASIS.PERCENT,
      priorityHysteresis: 3,
    };
    const values: PriorityValues = {
      ...serverValues,
      priorityStrategy: PRIORITY_STRATEGY.NONE,
      priorityHysteresis: "",
    };

    expect(changedPrioritySettings(values, serverValues)).toEqual(["priorityStrategy"]);
  });

  test("keeps hysteresis changes while the strategy is active", () => {
    const serverValues: PriorityValues = {
      priorityStrategy: PRIORITY_STRATEGY.SOC,
      priorityBasis: PRIORITY_BASIS.PERCENT,
      priorityHysteresis: 3,
    };
    const values: PriorityValues = { ...serverValues, priorityHysteresis: 5 };

    expect(changedPrioritySettings(values, serverValues)).toEqual(["priorityHysteresis"]);
  });
});
