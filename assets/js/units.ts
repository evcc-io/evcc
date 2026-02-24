import settings from "./settings";
import { LENGTH_UNIT } from "./types/evcc";

const MILES_FACTOR = 0.6213711922;

function isMiles() {
  return settings.unit === LENGTH_UNIT.MILES;
}

export function distanceValue(value: number) {
  return isMiles() ? value * MILES_FACTOR : value;
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
