import { GRID_CONNECTION } from "@/types/evcc";

// multiplier converting per-phase current times voltage into total power
function phaseFactor(phases = 0, gridConnection?: GRID_CONNECTION) {
  return gridConnection === GRID_CONNECTION.THREE_PHASE_THREE_WIRE && phases === 3
    ? Math.sqrt(3)
    : phases;
}

export function currentToPower(
  current = 0,
  phases = 0,
  voltage = 230,
  gridConnection?: GRID_CONNECTION
) {
  return voltage * current * phaseFactor(phases, gridConnection);
}

export function powerToCurrent(
  power = 0,
  phases = 0,
  voltage = 230,
  gridConnection?: GRID_CONNECTION
) {
  return power / (voltage * phaseFactor(phases, gridConnection));
}
