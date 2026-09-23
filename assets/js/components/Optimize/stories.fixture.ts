import type { EvoptData } from "./TimeSeriesDataTable.vue";
import type { BatteryDetail, DemandDetail } from "@/types/evcc";

// 15-minute slots from 06:00 to 22:00, energy per slot in Wh
const SLOT_H = 0.25;
const START_H = 6;
const SLOTS = 64;
const dt = Array(SLOTS).fill(SLOT_H * 3600);
export const timestamps = Array.from({ length: SLOTS }, (_, i) =>
  new Date(2025, 5, 12, START_H, i * 15).toISOString()
);

const hours = Array.from({ length: SLOTS }, (_, i) => START_H + i * SLOT_H);
const wh = (kw: (h: number, i: number) => number) => hours.map((h, i) => kw(h, i) * SLOT_H * 1000);

// solar bell peaking around 13:00
const solar = wh((h) => 6.5 * Math.exp(-(((h - 13) / 3.2) ** 2)));
// household base load with a lunch and evening bump
export const home: DemandDetail = {
  type: "home",
  values: wh(
    (h) => 0.4 + 0.3 * Math.exp(-(((h - 12.5) / 1) ** 2)) + 0.6 * Math.exp(-(((h - 19) / 1.5) ** 2))
  ),
};
// heat pump tapering off after the morning
export const heatPump: DemandDetail = {
  type: "heating",
  title: "Wärmepumpe",
  values: wh((h) => 0.5 + 1.2 * Math.exp(-(h - START_H) / 3)),
};
// heating rod boosts hot water twice a day
export const heatingRod: DemandDetail = {
  type: "heating",
  title: "Warmwasser",
  values: wh((h) => ((h >= 11 && h < 12) || (h >= 17 && h < 18) ? 2.2 : 0)),
};
// measured carport power decaying over the first slots, no forecast beyond
export const carport: DemandDetail = {
  type: "unmodelled",
  title: "Carport",
  values: wh((_, i) => Math.max(0, 6.9 * (1 - i / 4))),
};
// vehicle charges from surplus through the solar peak
const vehicle = wh((h) => (h >= 11 && h < 15 ? 5 : 0));

// palette colors aligned with the demand details, base load stays muted
export const demandColors = ["", "#F8961E", "#00B4D8"];
export const batteryColors = ["#0077B6", "#40A578"];

export const batteryDetails: BatteryDetail[] = [
  { type: "vehicle", title: "Garage (ID.3)", name: "lp-1", capacity: 58 },
  { type: "battery", title: "Home Battery", name: "battery", capacity: 10 },
];

// gt is the sum of the profiles
export function evopt(profiles: DemandDetail[]): EvoptData {
  const gt = hours.map((_, i) => profiles.reduce((sum, p) => sum + (p.values[i] || 0), 0));

  // home battery balances the residual within its limits, the grid takes the rest
  const limit = 2.5 * SLOT_H * 1000;
  const capacity = 10000;
  const charging: number[] = [];
  const discharging: number[] = [];
  const gridImport: number[] = [];
  const gridExport: number[] = [];
  const soc: number[] = [];
  let energy = 3000;
  hours.forEach((_, i) => {
    const residual = solar[i]! - gt[i]! - vehicle[i]!;
    const battery = Math.max(-limit, -energy, Math.min(limit, capacity - energy, residual));
    energy += battery;
    charging.push(Math.max(0, battery));
    discharging.push(Math.max(0, -battery));
    gridExport.push(Math.max(0, residual - battery));
    gridImport.push(Math.max(0, battery - residual));
    soc.push((energy / capacity) * 100);
  });

  return {
    req: {
      time_series: {
        ft: solar,
        // prices per Wh, midday dip
        p_E: Array(SLOTS).fill(0.000084),
        p_N: hours.map((h) => (0.31 - 0.12 * Math.exp(-(((h - 13) / 3) ** 2))) / 1000),
        gt,
        dt,
      },
    },
    res: {
      grid_import: gridImport,
      grid_export: gridExport,
      batteries: [
        {
          charging_power: vehicle,
          discharging_power: Array(SLOTS).fill(0),
          state_of_charge: hours.map((h) => Math.min(80, 40 + Math.max(0, h - 11) * 10)),
        },
        { charging_power: charging, discharging_power: discharging, state_of_charge: soc },
      ],
    },
  };
}
