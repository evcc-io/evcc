import * as echarts from "echarts/core";
import colors from "@/colors";
import type { ForecastSlot } from "./types";
import { BarChart, LineChart } from "echarts/charts";
import {
  GridComponent,
  TooltipComponent,
  MarkPointComponent,
  AxisPointerComponent,
} from "echarts/components";
import { SVGRenderer } from "echarts/renderers";

echarts.use([
  BarChart,
  LineChart,
  GridComponent,
  TooltipComponent,
  AxisPointerComponent,
  MarkPointComponent,
  SVGRenderer,
]);

export const FONT_FAMILY = "Montserrat, sans-serif";

export function markPointLabel(
  color: string,
  data: { coord: [string, number]; value: string; label?: { offset?: [number, number] } }[],
  startDate?: Date,
  endDate?: Date
) {
  // threshold: points within the first 5% of the time range get shifted right
  const threshold =
    startDate && endDate
      ? startDate.getTime() + (endDate.getTime() - startDate.getTime()) * 0.05
      : 0;

  const adjustedData = data.map((d) => {
    const ts = new Date(d.coord[0]).getTime();
    if (threshold && ts < threshold) {
      const baseOffset = d.label?.offset ?? [0, -2];
      return {
        ...d,
        label: {
          ...d.label,
          offset: [baseOffset[0] + 30, baseOffset[1]] as [number, number],
        },
      };
    }
    return d;
  });

  return {
    animation: true,
    clip: true,
    symbol: "rect",
    symbolSize: 1,
    label: {
      show: true,
      position: "top",
      fontFamily: FONT_FAMILY,
      fontSize: 14,
      fontWeight: "bold" as const,
      color: colors.background,
      backgroundColor: color,
      borderRadius: 4,
      padding: [5, 10],
      offset: [0, -2],
      formatter: (p: { data: { value: string } }) => p.data.value,
    },
    data: adjustedData,
  };
}

export function tooltipStyle(
  color: string,
  getChart?: () => { convertToPixel: echarts.ECharts["convertToPixel"] } | null
) {
  return {
    confine: true,
    backgroundColor: color,
    borderColor: color,
    borderWidth: 0,
    padding: [5, 10],
    extraCssText: "box-shadow: none; border-radius: 4px; text-align: center; z-index: 1000;",
    position(
      point: [number, number],
      params: { value: [string, number] }[] | { value: [string, number] },
      el: HTMLElement
    ): [number, number] {
      const w = el?.offsetWidth || 0;
      const h = el?.offsetHeight || 0;
      const margin = 8;
      const arr = Array.isArray(params) ? params : [params];
      const p = arr[0];
      const chart = getChart?.();
      if (chart && p?.value) {
        const pixelY = chart.convertToPixel({ seriesIndex: 0 }, p.value)?.[1];
        if (pixelY != null) {
          return [point[0] - w / 2, pixelY - h - margin];
        }
      }
      return [point[0] - w / 2, point[1] - h - margin];
    },
    textStyle: {
      fontFamily: FONT_FAMILY,
      fontSize: 14,
      fontWeight: "bold" as const,
      color: colors.background,
    },
  };
}

export function forecastGrid() {
  return { top: 36, right: 16, bottom: 4, left: 34, borderWidth: 0 };
}

export function forecastXAxes(startDate: Date, endDate: Date, weekdayShort: (d: Date) => string) {
  return [
    {
      type: "time",
      min: startDate,
      max: endDate,
      minInterval: 3600 * 1000,
      maxInterval: 3600 * 1000,
      axisLabel: {
        color: colors.muted,
        fontSize: 14,
        lineHeight: Math.round(14 * 1.1),
        margin: 4,
        formatter: (value: number) => {
          const date = new Date(value);
          const h = date.getHours();
          if (h % 4 !== 0) return "";
          if (h === 0) return `${h}\n${weekdayShort(date)}`;
          return `${h}`;
        },
      },
      splitLine: { show: false },
      axisLine: { show: false },
      axisTick: { show: false },
    },
    {
      type: "time",
      position: "bottom",
      min: startDate,
      max: endDate,
      minInterval: 24 * 3600 * 1000,
      maxInterval: 24 * 3600 * 1000,
      axisLabel: { show: false },
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: {
        show: true,
        showMinLine: false,
        showMaxLine: false,
        lineStyle: { color: colors.border || "", type: "dashed" },
      },
    },
  ];
}

export function forecastYAxis(overrides: Record<string, unknown> = {}) {
  const { axisLabel, ...rest } = overrides;
  return {
    type: "value",
    min: 0,
    axisLine: { show: false },
    axisTick: { show: false },
    splitLine: {
      showMinLine: false,
      showMaxLine: false,
      lineStyle: { color: colors.border || "" },
    },
    axisLabel: {
      fontSize: 10,
      opacity: 0.5,
      ...(axisLabel as Record<string, unknown>),
    },
    ...rest,
  };
}

export function clampStart(ts: string, startDate: Date): string {
  return new Date(ts) < startDate ? startDate.toISOString() : ts;
}

export function filterForecastSlots(
  slots: ForecastSlot[],
  startDate: Date,
  endDate: Date
): ForecastSlot[] {
  if (!Array.isArray(slots)) return [];
  return slots.filter((s) => new Date(s.end) > startDate && new Date(s.start) <= endDate);
}

export function minSlotIndex(slots: ForecastSlot[]): number {
  return slots.reduce((min, s, i) => (s.value < (slots[min]?.value || Infinity) ? i : min), 0);
}

export function maxSlotIndex(slots: ForecastSlot[]): number {
  return slots.reduce((max, s, i) => (s.value > (slots[max]?.value || 0) ? i : max), 0);
}

export { echarts };
