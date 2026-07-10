import settings from "./settings";
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

// short hour label for chart axes: "14" or "2 PM"
export function fmtHourShort(h: number) {
  return is12hFormat() ? `${h % 12 || 12} ${h < 12 ? "AM" : "PM"}` : String(h);
}
