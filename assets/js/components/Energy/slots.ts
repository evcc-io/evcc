// the metrics are recorded per 15 minute slot
export const SLOT_MS = 15 * 60 * 1000;

// average power in W of a slot's energy in kWh
export const slotWatts = (kWh: number) => (kWh * 1000 * 36e5) / SLOT_MS;
