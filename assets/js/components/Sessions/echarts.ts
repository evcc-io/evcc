import colors from "@/colors";
import "./roundChart.css";
import { echarts, FONT_FAMILY, tooltipStyle, tooltipTable } from "../Forecast/echarts";
import { PieChart, RadarChart } from "echarts/charts";
import { PolarComponent, RadarComponent } from "echarts/components";

// additional chart types used by the Sessions view
echarts.use([PieChart, RadarChart, PolarComponent, RadarComponent]);

export * from "../Forecast/echarts";

type ChartSize = { getWidth(): number; getHeight(): number } | null;

// tooltip centered horizontally, in the upper or lower fifth depending on the pointer
export function topBottomCenterPosition(getChart: () => ChartSize) {
  return (point: [number, number], _params: unknown, el: HTMLElement): [number, number] => {
    const w = getChart()?.getWidth() || 0;
    const h = getChart()?.getHeight() || 0;
    const isTop = point[1] > h / 2;
    const y = (isTop ? h / 5 : h - h / 5) - (el?.offsetHeight || 0) / 2;
    return [(w - (el?.offsetWidth || 0)) / 2, y];
  };
}

// rounded caps for stacked bars: the topmost segment above 2% of the tallest
// bar carries the cap; smaller slivers above it are hidden (they would float
// as a detached hairline) and only appear in tooltips
export function roundedStackData(datasets: { data: (number | null)[] }[], count: number) {
  const totals = Array.from({ length: count }, (_, idx) =>
    datasets.reduce((sum, d) => sum + (d.data[idx] || 0), 0)
  );
  const threshold = Math.max(0, ...totals) * 0.02;
  const topPerSlot = totals.map((_, idx) => {
    for (let i = datasets.length - 1; i >= 0; i--) {
      if ((datasets[i]!.data[idx] || 0) > threshold) return i;
    }
    return -1;
  });
  return datasets.map((dataset, i) =>
    dataset.data.map((v, idx) => {
      if (topPerSlot[idx] === i) {
        return { value: v, itemStyle: { borderRadius: [10, 10, 0, 0] } };
      }
      if (topPerSlot[idx]! >= 0 && i > topPerSlot[idx]!) return null;
      return v;
    })
  );
}

// rings at top/4, top/2, top land at 50%, 71%, 100% radius on a sqrt scale
export function sqrtTicks(max: number): number[] {
  const magnitude = 10 ** Math.floor(Math.log10(max > 0 ? max : 1));
  const top = [1, 2, 4, 8, 10].find((n) => n * magnitude >= max)! * magnitude;
  return [top / 4, top / 2, top];
}

// polar bars plot sqrt(value) so the wedge area, not its radius, is proportional to the value
export function sqrtRadiusAxis(max: number, formatValue: (value: number) => string) {
  const ticks = sqrtTicks(max).map(Math.sqrt);
  return {
    min: 0,
    max: ticks[ticks.length - 1],
    axisLine: { show: false },
    axisTick: { show: false, customValues: ticks },
    splitLine: { lineStyle: { color: colors.border || "" } },
    axisLabel: {
      customValues: ticks,
      fontSize: 10,
      color: colors.muted || "",
      backgroundColor: colors.box || "",
      padding: [1, 3],
      formatter: (value: number) => formatValue(value * value),
    },
  };
}

// shared doughnut chart option, tooltip centered in the cutout
export function doughnutOption(
  data: { name: string; value: number; color: string }[],
  formatValue: (value: number) => string,
  getChart: () => ChartSize
) {
  return {
    animation: false,
    textStyle: { fontFamily: FONT_FAMILY },
    tooltip: {
      trigger: "item",
      ...tooltipStyle(colors.text || ""),
      position: (_point: [number, number], _params: unknown, el: HTMLElement) => [
        ((getChart()?.getWidth() || 0) - (el?.offsetWidth || 0)) / 2,
        ((getChart()?.getHeight() || 0) - (el?.offsetHeight || 0)) / 2,
      ],
      formatter: (params: { name: string; value: number }) =>
        tooltipTable(params.name, [{ values: [formatValue(params.value)] }]),
    },
    series: [
      {
        type: "pie",
        radius: ["70%", "95%"],
        itemStyle: {
          borderRadius: 10,
          borderWidth: 3,
          borderColor: colors.box || "",
        },
        label: { show: false },
        labelLine: { show: false },
        emphasis: { scale: false },
        data: data.map(({ name, value, color }) => ({
          name,
          value,
          itemStyle: { color },
        })),
      },
    ],
  };
}
