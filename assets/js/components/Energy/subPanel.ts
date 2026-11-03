import colors, { setAlpha } from "@/colors";
import { boundaryAxis, forecastYAxis, hoverDot, lineDefaults } from "./echarts";

// skinny panel below a chart's main plot, sharing its x axis
const PANEL_HEIGHT = 40;
const PANEL_GAP = 24;
export const PANEL_EXTRA = 48; // added chart height, the main plot gives up the rest

export interface SubPanel {
  series: Record<string, unknown>[];
  xAxis: Record<string, unknown>; // hidden, x axis 2, the series sit on it
  yAxis: Record<string, unknown>;
  track: string; // background of the panel, the full range
}

// a slot's value is taken at its start: points on the slot boundaries of a hidden x
// axis 2, with slot i centered as on the category axis. The last value is repeated to
// close the day
export function slotStarts(values: (number | null)[]): [number, number | null][] {
  return [
    ...values.map((v, i) => [i - 0.5, v] as [number, number | null]),
    [values.length - 0.5, values.at(-1) ?? null],
  ];
}

// labels at both ends of the range only, tight to fit into the grid's left inset
export function panelYAxis(min: number, max: number, format: (v: number) => string) {
  return forecastYAxis({
    gridIndex: 1,
    position: "left",
    min,
    max,
    interval: max - min,
    // the grid's background is the track, a second set of lines would fight the main grid
    splitLine: { show: false },
    axisLabel: { color: colors.muted || "", hideOverlap: true, margin: 4, formatter: format },
  });
}

// soc or temperature per category as line and area on grid 1
export function socTempPanel(
  values: (number | null)[],
  color: string,
  isTemp: boolean,
  format: (v: number) => string
): SubPanel {
  // soc spans 0 to 100, temperature the data's range widened to full tens
  let [min, max] = [0, 100];
  if (isTemp) {
    const present = values.filter((v): v is number => v !== null);
    min = Math.floor(Math.min(...present) / 10) * 10;
    max = Math.max(Math.ceil(Math.max(...present) / 10) * 10, min + 10);
  }
  // a light box needs less tint than a dark one
  const dark = document.documentElement.classList.contains("dark");
  return {
    // same hue as the area, so the fill stays a clean tint of the device color
    track: setAlpha(color, dark ? "1a" : "0d") || "",
    series: [
      {
        id: "soctemp",
        name: "soctemp",
        type: "line",
        xAxisIndex: 2,
        yAxisIndex: 1,
        data: slotStarts(values),
        smooth: true,
        ...hoverDot(color),
        connectNulls: true,
        lineStyle: { color, ...lineDefaults },
        areaStyle: { color: setAlpha(color, "40") },
        z: 4,
      },
    ],
    xAxis: { ...boundaryAxis(values.length), gridIndex: 1 },
    yAxis: panelYAxis(min, max, format),
  };
}

// the main plot's grid shortened, and the panel's grid below it
export function panelGrids(main: { bottom: number; left: number; right: number }, track: string) {
  return [
    { ...main, bottom: main.bottom + PANEL_HEIGHT + PANEL_GAP },
    {
      bottom: main.bottom,
      height: PANEL_HEIGHT,
      left: main.left,
      right: main.right,
      borderWidth: 0,
      show: true,
      backgroundColor: track,
    },
  ];
}
