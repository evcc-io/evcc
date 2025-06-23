import { SMART_COST_TYPE } from "@/types/evcc";

export default function (smartCostType?: SMART_COST_TYPE) {
  return smartCostType && Object.values(SMART_COST_TYPE).includes(smartCostType);
}
