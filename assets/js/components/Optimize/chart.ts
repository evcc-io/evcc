import { xAxisLabelStyle } from "../Forecast/echarts";
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

// category x axis over slot times, labels at full hours every 4h (6h on mobile)
export function slotXAxis(times: number[]) {
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
        return String(d.getHours());
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
