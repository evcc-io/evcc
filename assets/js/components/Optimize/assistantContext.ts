import type { EvOpt } from "@/types/evcc";

export const OPTIMIZER_DOCS = "https://github.com/evcc-io/optimizer";

interface Hour {
  start: Date;
  seconds: number;
  price: number; // currency, weighted by duration
  feedIn: number;
  pv: number; // Wh
  load: number;
  gridImport: number;
  gridExport: number;
  batteries: { charge: number; discharge: number; soc: number }[];
}

const pad = (n: number) => String(n).padStart(2, "0");

function hourLabel(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:00`;
}

const kwh = (wh: number) => (wh / 1000).toFixed(1);
const perKwh = (perWh: number) => (perWh * 1000).toFixed(3);

// planHours folds the slots of a plan into clock hours, energies summed, prices averaged by duration,
// soc taken at the end of the hour. Hourly slots map one to one.
export function planHours(evopt: EvOpt): Hour[] {
  const { dt, gt, ft, p_N, p_E } = evopt.req.time_series;
  const res: Hour[] = [];

  evopt.details.timestamp.forEach((ts, i) => {
    const start = new Date(ts);
    start.setMinutes(0, 0, 0);

    let hour = res[res.length - 1];
    if (!hour || hour.start.getTime() !== start.getTime()) {
      hour = {
        start,
        seconds: 0,
        price: 0,
        feedIn: 0,
        pv: 0,
        load: 0,
        gridImport: 0,
        gridExport: 0,
        batteries: evopt.res.batteries.map(() => ({ charge: 0, discharge: 0, soc: 0 })),
      };
      res.push(hour);
    }

    hour.seconds += dt[i] ?? 0;
    hour.price += (p_N[i] ?? 0) * (dt[i] ?? 0);
    hour.feedIn += (p_E[i] ?? 0) * (dt[i] ?? 0);
    hour.pv += ft[i] ?? 0;
    hour.load += gt[i] ?? 0;
    hour.gridImport += evopt.res.grid_import?.[i] ?? 0;
    hour.gridExport += evopt.res.grid_export?.[i] ?? 0;
    evopt.res.batteries.forEach((bat, b) => {
      const entry = hour.batteries[b]!;
      entry.charge += bat.charging_power[i] ?? 0;
      entry.discharge += bat.discharging_power[i] ?? 0;
      entry.soc = bat.state_of_charge[i] ?? entry.soc;
    });
  });

  for (const hour of res) {
    if (hour.seconds > 0) {
      hour.price /= hour.seconds;
      hour.feedIn /= hour.seconds;
    }
  }

  return res;
}

// planContext renders the plan as the assistant's page context: the parameters of every battery
// and one row per hour, so the model can reason over the whole horizon without paging through the
// raw arrays. Those stay reachable through the evopt state key.
export function planContext(
  evopt: EvOpt,
  currency: string,
  strategy: string,
  automatic: string
): string {
  const hours = planHours(evopt);
  const horizon = Math.round(hours.reduce((sum, h) => sum + h.seconds, 0) / 3600);
  const titles = evopt.details.batteryDetails.map((d) => d.title || d.name);

  const batteries = evopt.req.batteries.map((bat, i) => {
    const detail = evopt.details.batteryDetails[i];
    const extras = [
      bat.charge_from_grid ? "may charge from grid" : "no grid charging",
      bat.discharge_to_grid ? "may discharge to grid" : "no grid discharge",
      `remaining energy valued ${perKwh(bat.p_a)} ${currency}/kWh`,
      bat.p_demand?.some((v) => v > 0) ? "has charge demands" : "",
      bat.s_goal?.some((v) => v > 0) ? "has soc goals" : "",
    ].filter(Boolean);
    return `${i}: ${titles[i]} (${detail?.type}, ${detail?.capacity} kWh): soc ${kwh(bat.s_initial)} kWh within ${kwh(bat.s_min)}..${kwh(bat.s_max)} kWh, charge ${kwh(bat.c_min)}..${kwh(bat.c_max)} kW, discharge ${kwh(bat.d_max)} kW, ${extras.join(", ")}`;
  });

  const rows = hours.map((h) =>
    [
      hourLabel(h.start),
      perKwh(h.price),
      perKwh(h.feedIn),
      kwh(h.pv),
      kwh(h.load),
      `${kwh(h.gridImport)}/${kwh(h.gridExport)}`,
      ...h.batteries.map((b) => `${kwh(b.charge)}/${kwh(b.discharge)}/${kwh(b.soc)}`),
    ].join(" | ")
  );

  return [
    `The user is on the optimizer page, which shows the latest plan of the evcc optimizer.`,
    `How a plan is computed (cost objective, battery parameters, charging strategies) is documented at ${OPTIMIZER_DOCS}. Read it with fetchDocs when the question needs it, the sections "What it does" and "Levelling grid peaks" explain the model.`,
    `Plan from ${evopt.updated}: status ${evopt.res.status}, objective ${evopt.res.objective_value ?? "none"} ${currency}, ${evopt.req.time_series.dt.length} steps over ${horizon} h, charging strategy ${strategy || "none"}, automatic mode ${automatic}.`,
    `The complete request and response are in the state key evopt (.evopt.req, .evopt.res, .evopt.details), energies in Wh per step, prices in ${currency}/Wh.`,
    `Batteries:`,
    ...batteries,
    `Plan per hour, energies in kWh, prices in ${currency}/kWh, battery columns are charge/discharge/soc at the end of the hour:`,
    ["hour", "import price", "feed-in price", "pv", "load", "grid import/export", ...titles].join(
      " | "
    ),
    ...rows,
  ].join("\n");
}
