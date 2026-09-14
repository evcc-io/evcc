// Pure transforms that turn the raw /history/energy response and the optimizer result into
// the per-battery soc timelines the chart consumes. Kept framework-free for unit testing.

import type { BatteryMeter, EvOpt } from "@/types/evcc";
import type { SocPoint, BatterySeries, BatteryHistorySeries } from "./types";

// A soc timeline tagged with the key used to match it back to a battery device.
export interface KeyedSeries {
  key: string;
  points: SocPoint[];
}

// History is grouped per title (no device name), so key it by title, falling back to the group.
export function historyToSeries(raw: BatteryHistorySeries[]): KeyedSeries[] {
  return raw.map((s) => ({
    key: s.title || s.group,
    points: s.data
      .filter((d) => d.socTemp != null)
      .map((d) => ({ t: new Date(d.start).getTime(), soc: d.socTemp as number })),
  }));
}

// Optimizer soc per battery, keyed by the device name; only points at/after now, converted
// from Wh to percent using the battery capacity (kWh).
export function forecastToSeries(evopt: EvOpt | undefined, nowMs: number): KeyedSeries[] {
  if (!evopt?.res?.batteries || !evopt.details?.timestamp) return [];
  const ts = evopt.details.timestamp.map((t) => new Date(t).getTime());
  return evopt.res.batteries.flatMap((bat, i): KeyedSeries[] => {
    const det = evopt.details.batteryDetails?.[i];
    if (!det || det.type !== "battery" || !det.capacity) return [];
    const points: SocPoint[] = (bat.state_of_charge || [])
      .map((wh, j) => ({ t: ts[j]!, soc: (wh / (det.capacity * 1000)) * 100 }))
      .filter((p) => p.t >= nowMs && !Number.isNaN(p.soc));
    return [{ key: det.name, points }];
  });
}

// Merge per-device meta with its matched history (by title) and forecast (by device name).
export function buildChartBatteries(
  devices: BatteryMeter[],
  history: KeyedSeries[],
  forecast: KeyedSeries[]
): BatterySeries[] {
  return devices.map((d) => {
    const title = d.title || d.name || "";
    return {
      id: title,
      title,
      capacity: d.capacity || 0,
      currentSoc: d.soc,
      history: history.find((s) => s.key === title)?.points ?? [],
      forecast: forecast.find((s) => s.key === d.name)?.points ?? [],
    };
  });
}
