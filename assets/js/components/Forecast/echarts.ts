import * as echarts from "echarts/core";
import { BarChart, LineChart } from "echarts/charts";
import {
  GridComponent,
  TooltipComponent,
  MarkPointComponent,
  AxisPointerComponent,
} from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";

echarts.use([
  BarChart,
  LineChart,
  GridComponent,
  TooltipComponent,
  AxisPointerComponent,
  MarkPointComponent,
  CanvasRenderer,
]);

export const FONT_FAMILY = "Montserrat, sans-serif";

export function markPointLabel(
  color: string,
  data: { coord: [string, number]; value: string }[],
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
      const dl = d as Record<string, unknown>;
      const existing = (dl["label"] as Record<string, unknown>) || {};
      const baseOffset = (existing["offset"] as number[]) || [0, -2];
      return {
        ...d,
        label: {
          ...existing,
          offset: [(baseOffset[0] ?? 0) + 30, baseOffset[1] ?? -2],
        },
      };
    }
    return d;
  });

  return {
    animation: true,
    clip: false,
    symbol: "rect",
    symbolSize: 1,
    label: {
      show: true,
      position: "top",
      fontFamily: FONT_FAMILY,
      fontSize: 13,
      fontWeight: "bold" as const,
      color: "#fff",
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
  getChart?: () => { convertToPixel: echarts.ECharts["convertToPixel"] } | null,
) {
  return {
    backgroundColor: color,
    borderColor: color,
    borderWidth: 0,
    padding: [5, 10],
    extraCssText: "box-shadow: none; border-radius: 4px; text-align: center;",
    position(
      point: [number, number],
      params: { value: [string, number] }[] | { value: [string, number] },
      el: HTMLElement,
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
      fontSize: 13,
      fontWeight: "bold" as const,
      color: "#fff",
    },
  };
}

export { echarts };
