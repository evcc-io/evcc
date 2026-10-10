import { echarts } from "../Forecast/echarts";
import { weekStart } from "@/units";
import { SankeyChart, HeatmapChart, ThemeRiverChart } from "echarts/charts";
import { CalendarComponent, SingleAxisComponent, VisualMapComponent } from "echarts/components";

echarts.use([
  SankeyChart,
  HeatmapChart,
  ThemeRiverChart,
  CalendarComponent,
  SingleAxisComponent,
  VisualMapComponent,
]);

export * from "../Forecast/echarts";

// boundary axis positions between the last day of a week and the first of the next
export function weekBoundaries(from: Date, days: number): number[] {
  const start = weekStart();
  return Array.from({ length: days - 1 }, (_, i) => i + 1)
    .filter((i) => new Date(from.getFullYear(), from.getMonth(), i + 1).getDay() === start)
    .map((i) => i - 0.5);
}
