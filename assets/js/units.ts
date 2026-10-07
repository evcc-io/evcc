import settings from "./settings";
import type { DateFormat } from "./settings";
import { LENGTH_UNIT } from "./types/evcc";

const MILES_FACTOR = 0.6213711922;

function isMiles() {
  return settings.unit === LENGTH_UNIT.MILES;
}

export function distanceValue(value: number) {
  return isMiles() ? value * MILES_FACTOR : value;
}

export function distanceValueReverse(value: number) {
  return isMiles() ? value / MILES_FACTOR : value;
}

export function distanceUnit() {
  return isMiles() ? "mi" : "km";
}

export function getUnits() {
  return isMiles() ? LENGTH_UNIT.MILES : LENGTH_UNIT.KM;
}

export function setUnits(value: LENGTH_UNIT) {
  settings.unit = value;
}

export function is12hFormat() {
  return settings.is12hFormat;
}

export function set12hFormat(value: boolean) {
  settings.is12hFormat = value;
}

// first weekday in Date.getDay() numbering from the browser region, Monday if unsupported
export function weekStart(): number {
  try {
    // not in the TS lib yet
    const locale = new Intl.Locale(navigator.language) as {
      getWeekInfo?: () => { firstDay: number };
    };
    const day = locale.getWeekInfo?.().firstDay;
    return day ? day % 7 : 1;
  } catch {
    return 1;
  }
}

export function getDateFormat(): DateFormat {
  return settings.dateFormat || "";
}

export function setDateFormat(value: DateFormat) {
  settings.dateFormat = value;
}
