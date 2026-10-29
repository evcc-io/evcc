import { POWER_UNIT } from "../mixins/formatter";

export interface AxisScale {
  // below 1000 base units the axis shows the base unit (W, g) instead of the kilo unit
  small: boolean;
  digits: number;
  // nice-ceiled axis limit in base units
  limit: number;
}

export interface EnergyAxisScale {
  unit: POWER_UNIT.W | POWER_UNIT.KW;
  digits: number;
  // nice-ceiled axis limit in W(h)
  limit: number;
}

// Round up to a nice number (1/2/3/4/6/8 × 10^n).
export function niceCeil(v: number): number {
  if (v <= 0) return 0;
  const mag = Math.pow(10, Math.floor(Math.log10(v)));
  const r = v / mag;
  const n = r <= 1 ? 1 : r <= 2 ? 2 : r <= 3 ? 3 : r <= 4 ? 4 : r <= 6 ? 6 : r <= 8 ? 8 : 10;
  return n * mag;
}

// Shared y-axis scale for kilo-stepped units (kW, kWh, kg). Takes the data
// peak in base units: peaks below 1000 switch to the base unit with the axis
// floored at 1000 for stable context (zero data falls here too); kilo limits
// up to 3 get one decimal to avoid duplicate integer ticks.
export function axisScale(peak: number): AxisScale {
  const small = peak < 1000;
  const limit = small ? Math.max(niceCeil(peak), 1000) : niceCeil(peak);
  return { small, digits: !small && limit <= 3000 ? 1 : 0, limit };
}

export function energyAxisScale(peak: number): EnergyAxisScale {
  const { small, digits, limit } = axisScale(peak);
  return { unit: small ? POWER_UNIT.W : POWER_UNIT.KW, digits, limit };
}
