import settings from "./settings";

const NONE = "none";
const PRICE = "price";
const CO2 = "co2";
export const GRID_DETAILS = [NONE, PRICE, CO2];

export function getGridDetails() {
  return GRID_DETAILS.includes(settings.gridDetails) ? settings.gridDetails : NONE;
}

export function showGridPrice() {
  return settings.gridDetails === PRICE;
}

export function showGridCo2() {
  return settings.gridDetails === CO2;
}

export function setGridDetails(value) {
  settings.gridDetails = value;
}
