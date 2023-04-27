import settings from "./settings";

const KM = "km";
const MILES = "mi";
export const UNITS = [KM, MILES];

export const CO2_UNIT = "gCO2eq";

const MILES_FACTOR = 0.6213711922;

function isMiles() {
  return settings.unit === MILES;
}

export function distanceValue(value) {
  return isMiles() ? value * MILES_FACTOR : value;
}

export function distanceUnit() {
  return isMiles() ? "mi" : "km";
}

export function getUnits() {
  return isMiles() ? MILES : KM;
}

export function setUnits(value) {
  settings.unit = value;
}
