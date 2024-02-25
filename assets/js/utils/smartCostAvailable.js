import { CO2_TYPE, PRICE_DYNAMIC_TYPE, PRICE_FORECAST_TYPE } from "../units";

export default function (smartCostType) {
  return [CO2_TYPE, PRICE_DYNAMIC_TYPE, PRICE_FORECAST_TYPE].includes(smartCostType);
}
