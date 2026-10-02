import type { Flow, FlowSink, FlowSource } from "./types";

export const SOURCES: FlowSource[] = ["pv", "battery", "grid"];

export function sumEnergy(
  series: { data: { energy: number; returnEnergy: number }[] },
  key: "energy" | "returnEnergy" = "energy"
): number {
  return series.data.reduce((acc, slot) => acc + slot[key], 0);
}

// share in percent of each source in what reached the sink over the period
export function sourceShares(flows: Flow[], sink: FlowSink): Record<FlowSource, number> {
  const into = flows.filter((f) => f.to === sink);
  const total = into.reduce((acc, f) => acc + f.energy, 0);
  const share = (from: FlowSource) =>
    total
      ? (into.filter((f) => f.from === from).reduce((a, f) => a + f.energy, 0) / total) * 100
      : 0;
  return { pv: share("pv"), battery: share("battery"), grid: share("grid") };
}
