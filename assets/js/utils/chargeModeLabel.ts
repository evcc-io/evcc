import { CHARGE_MODE } from "@/types/evcc";

// i18n key for a charge mode, device-aware (heating: normal/boost, switch: on)
export default function chargeModeLabelKey(
  mode: CHARGE_MODE,
  continuous: boolean,
  switchDevice: boolean
): string {
  if (mode === CHARGE_MODE.OFF && continuous) {
    return "main.mode.normal";
  }
  if (mode === CHARGE_MODE.NOW) {
    if (continuous) {
      return "main.mode.boost";
    }
    if (switchDevice) {
      return "main.mode.on";
    }
  }
  return `main.mode.${mode}`;
}
