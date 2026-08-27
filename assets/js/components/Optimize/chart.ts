import { xAxisLabelStyle } from "../Forecast/echarts";
import colors from "@/colors";
import type { BatteryDetail } from "@/types/evcc";

// loadpoint part of a vehicle entry title: "Carport (blue e-Golf)" → "Carport"
export function loadpointTitle(detail: BatteryDetail): string {
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
// day boundaries as split line with the weekday shown at 00:00
export function slotXAxis(times: number[], weekdayShort: (d: Date) => string) {
  const step = window.innerWidth < 576 ? 6 : 4;
  return {
    type: "category",
    data: times.map(String),
    axisLine: { show: false },
    axisTick: { show: false },
    splitLine: {
      show: true,
      // split lines sit at the left edge of their slot
      interval: (index: number) => index > 0 && isMidnight(times[index]),
      lineStyle: { color: colors.muted || "", type: "solid" },
    },
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

// slot energy (Wh) to average power (kW)
export function whToKW(wh: number, dtSeconds: number): number {
  return wh / (dtSeconds / 3600) / 1000;
}
