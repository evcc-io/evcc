import type { TimeseriesEntry, SolarDetails } from "../components/Forecast/types";
import deepCopy from "./deepClone";

export enum ForecastType {
  Solar = "solar",
  Price = "price",
  Co2 = "co2",
}

// return the date in local YYYY-MM-DD format
function toDayString(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

// return only slots that are on a given date, ignores slots that are in the past
export function filterEntriesByDate(
  entries: TimeseriesEntry[],
  dayString: string
): TimeseriesEntry[] {
  const now = new Date();
  return entries.filter(({ ts }) => {
    const isPast = new Date(ts) < now;
    const dateMatches = toDayString(new Date(ts)) === dayString;
    return !isPast && dateMatches;
  });
}

// return the date in local YYYY-MM-DD format
export function dayStringByOffset(day: number): string {
  const date = new Date();
  date.setDate(date.getDate() + day);
  return toDayString(date);
}

// return the highest slot for a given day (0 = today, 1 = tomorrow, etc.)
export function highestSlotIndexByDay(entries: TimeseriesEntry[], day: number = 0): number {
  const dayString = dayStringByOffset(day);
  const dayEntries = filterEntriesByDate(entries, dayString);
  const sortedEntries = dayEntries.sort((a, b) => b.val - a.val);
  const highestEntry = sortedEntries[0];
  if (!highestEntry) return -1;
  return entries.findIndex((entry) => entry.ts === highestEntry.ts);
}

export function adjustedSolar(solar?: SolarDetails): SolarDetails | undefined {
  if (!solar?.scale) return solar;

  const { scale } = solar;
  const result = deepCopy(solar);

  if (result.today) result.today.energy *= scale;
  if (result.tomorrow) result.tomorrow.energy *= scale;
  if (result.dayAfterTomorrow) result.dayAfterTomorrow.energy *= scale;
  if (result.timeseries) {
    result.timeseries.forEach((entry) => {
      entry.val *= scale;
    });
  }

  result.scale = 1 / scale; // invert to allow back-adjustment

  return result;
}

export interface SlotWithValue {
  start: string | Date;
  value: number;
}

// find index of first slot with lowest sum over span (e.g. span=4 for 1h with 15min slots)
export function findLowestSumSlotIndex(slots: SlotWithValue[], span: number): number {
  if (slots.length < span) return -1;

  let lowestSum = Infinity;
  let lowestIndex = -1;

  for (let i = 0; i <= slots.length - span; i++) {
    const sum = slots.slice(i, i + span).reduce((acc, slot) => acc + slot.value, 0);
    if (sum < lowestSum) {
      lowestSum = sum;
      lowestIndex = i;
    }
  }

  return lowestIndex;
}
