import * as echarts from "echarts/core";
import colors from "@/colors";
import escapeHtml from "@/utils/escapeHtml";
import type { UiForecastSlot } from "@/types/evcc";
import { BarChart, LineChart } from "echarts/charts";
import {
  GridComponent,
  TooltipComponent,
  MarkPointComponent,
  MarkLineComponent,
  MarkAreaComponent,
  GraphicComponent,
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
  MarkLineComponent,
  MarkAreaComponent,
  GraphicComponent,
  SVGRenderer,
]);

export const FONT_FAMILY = "Montserrat, sans-serif";

export function markPointLabel(
  color: string,
  data: { coord: [number, number]; value: string; label?: { offset?: [number, number] } }[],
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
    symbolSize: 0,
    label: {
      show: true,
      position: "top",
      fontFamily: FONT_FAMILY,
      fontSize: 14,
      fontWeight: "bold" as const,
      color: colors.background,
      backgroundColor: color,
      borderRadius: 4,
      padding: [5, 15],
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

export interface TooltipRow {
  name?: string;
  values: string[];
  total?: boolean;
}

// Shared tooltip: bold date headline, non-bold rows, optional name + value columns.
export function tooltipTable(head: string, rows: TooltipRow[], headers?: string[]): string {
  const hasName = rows.some((r) => r.name != null);
  const valueCols = Math.max(1, ...rows.map((r) => r.values.length));
  const colCount = (hasName ? 1 : 0) + valueCols;
  const valClsFn = (i: number): string => {
    // first of two unnamed value cols
    if (!hasName && valueCols > 1 && i === 0) return "text-start";
    // named or multiple cols
    if (hasName || valueCols > 1) return "text-end ps-3";
    // lone value
    return "text-center";
  };
  const headerRow = headers?.length
    ? `<tr>${hasName ? "<td></td>" : ""}${headers
        .map((h, i) => `<td class="fw-normal tabular ${valClsFn(i)}">${h}</td>`)
        .join("")}</tr>`
    : "";
  const rowHtml = (r: TooltipRow) => {
    const cls = r.total ? " pt-1" : "";
    const nameTd = hasName
      ? `<td class="fw-normal text-start${cls}">${escapeHtml(r.name ?? "")}</td>`
      : "";
    const valTds = r.values
      .map((v, i) => `<td class="fw-normal tabular ${valClsFn(i)}${cls}">${v}</td>`)
      .join("");
    return `<tr>${nameTd}${valTds}</tr>`;
  };
  const body = rows
    .filter((r) => !r.total)
    .map(rowHtml)
    .join("");
  const footRows = rows.filter((r) => r.total);
  const foot = footRows.length
    ? `<tfoot><tr><td colspan="${colCount}" class="pt-1 border-bottom"></td></tr>${footRows
        .map(rowHtml)
        .join("")}</tfoot>`
    : "";
  return `<table class="lh-sm"><thead><tr><th colspan="${colCount}" class="fw-bold text-center pb-1">${head}</th></tr></thead><tbody>${headerRow}${body}</tbody>${foot}</table>`;
}

export function forecastGrid() {
  return { top: 36, right: 16, bottom: 16, left: 24, borderWidth: 0 };
}

// common x-axis label styling across time-based charts
export function xAxisLabelStyle() {
  return {
    color: colors.muted || "",
    fontSize: 14,
    lineHeight: Math.round(14 * 1.1),
    margin: 4,
  };
}

export function forecastXAxes(
  startDate: Date | number,
  endDate: Date | number,
  hourShort: (d: Date) => string,
  weekdayShort: (d: Date) => string,
  stepHours = 4
) {
  return [
    {
      type: "time",
      min: startDate,
      max: endDate,
      minInterval: 3600 * 1000,
      maxInterval: 3600 * 1000,
      axisLabel: {
        ...xAxisLabelStyle(),
        hideOverlap: false,
        formatter: (value: number) => {
          const date = new Date(value);
          const h = date.getHours();
          if (h % stepHours !== 0) return "";
          const label = hourShort(date);
          if (h === 0) return `${label}\n${weekdayShort(date)}`;
          return label;
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
      showMinLine: true,
      showMaxLine: false,
      lineStyle: { color: colors.border || "" },
    },
    axisLabel: {
      fontSize: 10,
      opacity: 0.75,
      ...(axisLabel as Record<string, unknown>),
    },
    ...rest,
  };
}

export function clampStart(ts: number, startDate: Date): number {
  return Math.max(ts, startDate.getTime());
}

export function filterForecastSlots(
  slots: UiForecastSlot[],
  startDate: Date,
  endDate: Date
): UiForecastSlot[] {
  if (!Array.isArray(slots)) return [];
  return slots.filter((s) => new Date(s.end) > startDate && new Date(s.start) <= endDate);
}

export function minSlotIndex(slots: UiForecastSlot[]): number {
  return slots.reduce((min, s, i) => (s.value < (slots[min]?.value ?? Infinity) ? i : min), 0);
}

export function maxSlotIndex(slots: UiForecastSlot[]): number {
  return slots.reduce((max, s, i) => (s.value > (slots[max]?.value || 0) ? i : max), 0);
}

export { echarts };
