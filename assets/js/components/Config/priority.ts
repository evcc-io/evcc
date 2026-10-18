import { PRIORITY_BASIS, PRIORITY_STRATEGY } from "@/types/evcc";

export interface PriorityValues {
  priorityStrategy: PRIORITY_STRATEGY;
  priorityBasis: PRIORITY_BASIS;
  priorityHysteresis: number | string;
}

export const changedPrioritySettings = (
  values: PriorityValues,
  serverValues: PriorityValues
): (keyof PriorityValues)[] => {
  return (Object.keys(values) as (keyof PriorityValues)[]).filter((key) => {
    if (key === "priorityHysteresis" && values.priorityStrategy === PRIORITY_STRATEGY.NONE) {
      return false;
    }
    return values[key] !== serverValues[key];
  });
};
