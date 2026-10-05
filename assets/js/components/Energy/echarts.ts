import { echarts } from "../Forecast/echarts";
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
