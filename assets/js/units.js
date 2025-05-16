import settings from "./settings";

const KM = "km";
const MILES = "mi";
export const UNITS = [KM, MILES];

export const CO2_TYPE = "co2";
export const PRICE_DYNAMIC_TYPE = "pricedynamic";
export const PRICE_FORECAST_TYPE = "priceforecast";

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

export function is12hFormat() {
  return settings.is12hFormat;
}

export function set12hFormat(value) {
  settings.is12hFormat = value;
}
