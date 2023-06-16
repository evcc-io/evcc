import settings from "./settings";

const TIME = "time";
const CO2 = "co2";
const PRICE = "price";
const AVG_PRICE = "avgPrice";
const SOLAR = "solar";
export const SESSION = {
  TIME,
  AVG_PRICE,
  PRICE,
  SOLAR,
  CO2,
};

export function getSessionInfo(index) {
  return settings.sessionInfo[index - 1] || TIME;
}

export function setSessionInfo(index, value) {
  const clone = [...settings.sessionInfo];
  clone[index - 1] = value;
  clone.map((v) => v || "");
  settings.sessionInfo = clone;
}
