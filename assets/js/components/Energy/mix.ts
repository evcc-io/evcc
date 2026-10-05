import type { Flow, FlowSink, FlowSource } from "./types";

export function sumEnergy(
  series: { data: { energy: number; returnEnergy: number }[] },
  key: "energy" | "returnEnergy" = "energy"
): number {
  return series.data.reduce((acc, slot) => acc + slot[key], 0);
}

// what the energy into the given sinks cost, and how much of it was priced
export function sinkCost(flows: Flow[], ...sinks: FlowSink[]): { cost: number; energy: number } {
  const into = flows.filter((f) => sinks.includes(f.to));
  return {
    cost: into.reduce((acc, f) => acc + (f.cost ?? 0), 0),
    energy: into.reduce((acc, f) => acc + (f.pricedEnergy ?? 0), 0),
  };
}

export interface SourceRow {
  from: FlowSource;
  share: number; // percent of what reached the sink
  energy: number; // kWh
  cost?: number; // undefined without price data
  price?: number; // per kWh
}

// what reached the sink from each of `sources` over the period, scaled to `energy`:
// an entity of the sink (one consumer, one loadpoint) inherits the sink's split and prices
export function sourceBreakdown(
  flows: Flow[],
  sink: FlowSink,
  energy: number,
  sources: FlowSource[]
): SourceRow[] {
  const into = flows.filter((f) => f.to === sink);
  const total = into.reduce((acc, f) => acc + f.energy, 0);
  if (!total) return [];
  return sources.map((from) => {
    const flow = into.find((f) => f.from === from);
    const share = (flow?.energy ?? 0) / total;
    const price = flow?.pricedEnergy ? (flow.cost ?? 0) / flow.pricedEnergy : undefined;
    return {
      from,
      share: share * 100,
      energy: share * energy,
      price,
      cost: price === undefined ? undefined : price * share * energy,
    };
  });
}
