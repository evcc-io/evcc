// Combined battery soc from the history series, shared by the chart line and the legend.

import type { BatteryMeter } from "@/types/evcc";
import type { HistorySeries } from "./GroupChart.vue";

// Configured capacity per battery, keyed like the history series title.
export function batteryCapacities(devices: BatteryMeter[]): Map<string, number> {
  const res = new Map<string, number>();
  for (const d of devices) {
    res.set(d.title || d.name || "", d.capacity || 0);
  }
  return res;
}

// Combined soc per slot start: stored energy over total capacity. Empty when a
// capacity is missing, slots without soc for every battery are omitted.
export function totalSocByStart(
  series: HistorySeries[],
  capacities: Map<string, number>
): Map<string, number> {
  const res = new Map<string, number>();
  if (!series.length || series.some((s) => !capacities.get(s.title))) return res;

  const acc = new Map<string, { stored: number; capacity: number; count: number }>();
  for (const s of series) {
    const capacity = capacities.get(s.title) || 0;
    for (const slot of s.data) {
      if (slot.socTemp == null) continue;
      const e = acc.get(slot.start) ?? { stored: 0, capacity: 0, count: 0 };
      e.stored += (slot.socTemp / 100) * capacity;
      e.capacity += capacity;
      e.count++;
      acc.set(slot.start, e);
    }
  }

  for (const [start, e] of acc) {
    if (e.count === series.length && e.capacity > 0) {
      res.set(start, (e.stored / e.capacity) * 100);
    }
  }
  return res;
}
