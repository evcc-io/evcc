import { echarts, forecastGrid } from "../Forecast/echarts";
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

// same insets on every energy plot so they line up across cards, wide enough for the
// value axis on the right and the panel axis or side labels on the left
export function energyGrid() {
  return { ...forecastGrid(), left: 36, right: 36 };
}

// boundary axis positions between the last day of a week and the first of the next
export function weekBoundaries(from: Date, days: number, country?: string): number[] {
  const start = weekStart(country);
  return Array.from({ length: days - 1 }, (_, i) => i + 1)
    .filter((i) => new Date(from.getFullYear(), from.getMonth(), i + 1).getDay() === start)
    .map((i) => i - 0.5);
}
