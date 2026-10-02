import { echarts } from "../Forecast/echarts";
import { SankeyChart, HeatmapChart, CustomChart } from "echarts/charts";
import { CalendarComponent, VisualMapComponent } from "echarts/components";

echarts.use([SankeyChart, HeatmapChart, CustomChart, CalendarComponent, VisualMapComponent]);

export * from "../Forecast/echarts";
