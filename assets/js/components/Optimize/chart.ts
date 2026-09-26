import { hoverDot, xAxisLabelStyle } from "../Forecast/echarts";
import colors from "@/colors";
import type { DemandDetail } from "@/types/evcc";

export type Titled = { title?: string; name?: string };

// base load carries no title, loadpoint profiles are named after the loadpoint
export function demandTitle(detail: DemandDetail, baseLoadLabel: string): string {
  if (detail.type === "home") return baseLoadLabel;
  return `${detail.title} (${detail.type})`;
}

// loadpoint part of a vehicle entry title: "Carport (blue e-Golf)" → "Carport"
export function loadpointTitle(detail: Titled): string {
  const title = detail.title || detail.name || "";
  return title.replace(/\s*\([^)]*\)$/, "");
}

// slot start times (ms) from the first timestamp plus cumulative dt seconds
export function slotTimes(timestamp: string, dt: number[]): number[] {
  const start = new Date(timestamp).getTime();
  const res: number[] = [];
  let acc = 0;
  for (const s of dt) {
    res.push(start + acc * 1000);
    acc += s;
  }
  return res;
}

export function isMidnight(time?: number): boolean {
  if (time === undefined) return false;
  const d = new Date(time);
  return d.getHours() === 0 && d.getMinutes() === 0;
}

// category x axis over slot times, labels at full hours every 4h (6h on mobile),
// weekday shown at 00:00
export function slotXAxis(times: number[], weekdayShort: (d: Date) => string) {
  const step = window.innerWidth < 576 ? 6 : 4;
  return {
    type: "category",
    data: times.map(String),
    axisLine: { show: false },
    axisTick: { show: false },
    splitLine: { show: false },
    axisLabel: {
      ...xAxisLabelStyle(),
      interval: 0,
      formatter: (value: string) => {
        const d = new Date(Number(value));
        if (d.getMinutes() !== 0 || d.getHours() % step !== 0) return "";
        const label = String(d.getHours());
        return d.getHours() === 0 ? `${label}\n${weekdayShort(d)}` : label;
      },
    },
  };
}

export function formatSlotRange(times: number[], dt: number[], index: number): string {
  const start = new Date(times[index] ?? 0);
  const end = new Date(start.getTime() + (dt[index] || 0) * 1000);
  const f = (d: Date) =>
    `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
  return `${f(start)} - ${f(end)}`;
}

// hover dot created on demand: a hidden symbol per slot makes the svg renderer re-insert all
// of them on every hover frame. The temporary dot only drops when new data re-renders the chart.
export function transientHoverDot(color: string) {
  return { ...hoverDot(color), showSymbol: false };
}

// slot energy (Wh) to average power (kW)
export function whToKW(wh: number, dtSeconds: number): number {
  return wh / (dtSeconds / 3600) / 1000;
}

// hidden value axis aligned with the category bands (slot i spans i-0.5..i+0.5)
export function dayBoundaryAxis(times: number[]) {
  return {
    type: "value",
    min: -0.5,
    max: times.length - 0.5,
    show: false,
    axisPointer: { show: false },
  };
}

// day boundary lines. A category split line would sit in the gap between bars
// and show as a sliver; this draws them just inside the bar's left edge, behind the bars.
export function dayBoundarySeries(times: number[]) {
  const data = times
    .map((t, i) => ({ t, i }))
    .filter(({ t, i }) => i > 0 && isMidnight(t))
    .map(({ i }) => ({ xAxis: i - 0.15 }));
  return {
    type: "line",
    xAxisIndex: 1,
    data: [] as number[],
    silent: true,
    markLine: {
      silent: true,
      z: 1,
      symbol: "none",
      label: { show: false },
      lineStyle: { color: colors.muted || "", type: "solid" },
      data,
    },
  };
}
