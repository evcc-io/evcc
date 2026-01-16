import type { Rate, Slot } from "../types/evcc";

export function calculateCostRange(slots: Slot[]): {
  min: number | undefined;
  max: number | undefined;
} {
  let min = undefined as number | undefined;
  let max = undefined as number | undefined;
  slots.forEach((slot) => {
    if (slot.value === undefined) return;
    min = min === undefined ? slot.value : Math.min(min, slot.value);
    max = max === undefined ? slot.value : Math.max(max, slot.value);
  });
  return { min, max };
}

export function findRateInRange(start: Date, end: Date, rates: Rate[]): Rate | undefined {
  return rates.find((r) => {
    if (r.start.getTime() < start.getTime()) {
      return r.end.getTime() > start.getTime();
    }
    return r.start.getTime() < end.getTime();
  });
}

export function generateRateSlots(
  rates: Rate[],
  weekdayFormatter: (date: Date) => string,
  isCharging?: (value: number | undefined, start: Date, end: Date) => boolean,
  isWarning?: (value: number | undefined, start: Date, end: Date) => boolean
): Slot[] {
  if (!rates?.length) {
    return [];
  }

  const quarterHour = 15 * 60 * 1000;
  const base = new Date();
  base.setSeconds(0, 0);
  base.setMinutes(base.getMinutes() - (base.getMinutes() % 15));

  // Create slots for the duration of the rates
  const lastRate = rates[rates.length - 1]!;
  const duration = Math.ceil((lastRate.end.getTime() - base.getTime()) / quarterHour);

  return Array.from({ length: Math.min(duration, 96 * 4) }, (_, i) => {
    const start = new Date(base.getTime() + quarterHour * i);
    const end = new Date(start.getTime() + quarterHour);
    const value = findRateInRange(start, end, rates)?.value;

    const charging = isCharging ? isCharging(value, start, end) : false;
    const warning = isWarning ? isWarning(value, start, end) : false;
    const selectable = value !== undefined;

    return {
      day: weekdayFormatter(start),
      value,
      start,
      end,
      charging,
      selectable,
      warning,
    };
  });
}
