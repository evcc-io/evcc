import { OPTIMIZER_AUTOMATIC, type SelectOption } from "@/types/evcc";

type Translate = (key: string) => string;

export function automaticLevelOptions(t: Translate, enabled: boolean): SelectOption<string>[] {
  return Object.values(OPTIMIZER_AUTOMATIC).map((value) => ({
    value,
    name: t(`config.optimizer.automaticLevel.${value}`),
    disabled: !enabled,
  }));
}

// i18n keys for what the optimizer does at a level
export function automaticLevelActions(level: OPTIMIZER_AUTOMATIC): string[] {
  switch (level) {
    case OPTIMIZER_AUTOMATIC.BATTERY:
      return ["config.optimizer.automaticBattery"];
    case OPTIMIZER_AUTOMATIC.FULL:
      return [
        "config.optimizer.automaticBattery",
        "config.optimizer.automaticCharging",
        "config.optimizer.automaticLimits",
        "config.optimizer.automaticPlans",
      ];
    default:
      return [];
  }
}
