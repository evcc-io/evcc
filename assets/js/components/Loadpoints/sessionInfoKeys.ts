import type { SessionInfoKey } from "@/types/evcc";

// order in which stats fill up fixed slots as the screen gets wider
const PRIORITY: SessionInfoKey[] = [
  "duration",
  "solar",
  "price",
  "co2",
  "avgPrice",
  "remaining",
  "finished",
  "emission",
  "last24hEnergy",
  "last7dEnergy",
];

export interface SessionInfoAvailability {
  chargeRemainingDurationInterpolated?: number;
  tariffGrid?: number;
  tariffCo2?: number;
  last24hEnergy?: number;
  last7dEnergy?: number;
}

// keeps this in sync with the per-key `available` logic in SessionInfo.vue's
// `options` computed - used here only to decide slot counts, not formatting
export function availableSessionInfoKeys(props: SessionInfoAvailability): SessionInfoKey[] {
  const remainingAvailable = (props.chargeRemainingDurationInterpolated ?? 0) > 0;
  const availability: Record<SessionInfoKey, boolean> = {
    remaining: remainingAvailable,
    finished: remainingAvailable,
    duration: true,
    solar: true,
    avgPrice: props.tariffGrid !== undefined,
    price: props.tariffGrid !== undefined,
    co2: props.tariffCo2 !== undefined,
    emission: props.tariffCo2 !== undefined,
    last24hEnergy: props.last24hEnergy !== undefined,
    last7dEnergy: props.last7dEnergy !== undefined,
  };
  return PRIORITY.filter((key) => availability[key]);
}
