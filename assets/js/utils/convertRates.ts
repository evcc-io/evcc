import type { Rate } from "../types/evcc";
import type { ForecastSlot } from "../components/Forecast/types";

function convertRate(slot: ForecastSlot): Rate {
  return {
    start: new Date(slot.start),
    end: new Date(slot.end),
    value: slot.value,
  };
}

export default function convertRates(slots: ForecastSlot[] | null): Rate[] {
  if (!slots) return [];
  return slots.map(convertRate);
}
