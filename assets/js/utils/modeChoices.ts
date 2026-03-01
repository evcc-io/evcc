import { CHARGE_MODE } from "@/types/evcc";

export interface ChargeModeChoice {
  key: CHARGE_MODE;
  name: string;
}

interface ChargeModeChoicesOptions {
  pvPossible?: boolean;
  smartCostAvailable?: boolean;
  t?: (key: string) => string;
}

/**
 * Get available charge modes based on system capabilities.
 * Filters modes based on pvPossible and smartCostAvailable.
 */
function getAvailableChargeModes(
  pvPossible: boolean = false,
  smartCostAvailable: boolean = false
): CHARGE_MODE[] {
  if (pvPossible) {
    return [CHARGE_MODE.OFF, CHARGE_MODE.PV, CHARGE_MODE.MINPV, CHARGE_MODE.NOW];
  }
  if (smartCostAvailable) {
    return [CHARGE_MODE.OFF, CHARGE_MODE.PV, CHARGE_MODE.NOW];
  }
  return [CHARGE_MODE.OFF, CHARGE_MODE.NOW];
}

/**
 * Get mode label with smart renaming.
 * Renames 'pv' to 'smart' for non-pv and dynamic tariffs scenarios.
 */
function getChargeModeLabel(
  mode: CHARGE_MODE,
  t: (key: string) => string,
  pvPossible: boolean = false,
  smartCostAvailable: boolean = false
): string {
  // rename pv mode to smart for non-pv and dynamic tariffs scenarios
  if (mode === CHARGE_MODE.PV && !pvPossible && smartCostAvailable) {
    return t("main.mode.smart");
  }
  return t(`main.mode.${mode}`);
}

/**
 * Generate mode choices for select/choice fields.
 * Always returns a list of actual modes.  Callers that need a placeholder
 * must prepend their own entry ("{ key: '', name: '---' }").
 */
export function getChargeModeChoices(options: ChargeModeChoicesOptions = {}): ChargeModeChoice[] {
  const { pvPossible = false, smartCostAvailable = false, t = (key: string) => key } = options;

  const modes = getAvailableChargeModes(pvPossible, smartCostAvailable);

  return modes.map((mode) => ({
    key: mode,
    name: getChargeModeLabel(mode, t, pvPossible, smartCostAvailable),
  }));
}
