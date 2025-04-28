import { Chart, Tooltip } from "chart.js";
import colors from "@/colors";
// Register common components
export function registerChartComponents(components) {
  Chart.register(...components);
}

// Set default configurations immediately
Chart.defaults.font.family = window
  .getComputedStyle(document.documentElement)
  .getPropertyValue("--bs-font-sans-serif");
Chart.defaults.font.size = 14;
Chart.defaults.layout.padding = 0;

// Custom tooltip positioners
Tooltip.positioners.center = function () {
  const { chart } = this;
  return {
    x: chart.width / 2,
    y: chart.height / 2,
    xAlign: "center",
    yAlign: "center",
  };
};

Tooltip.positioners.topBottomCenter = function (context, eventPosition) {
  const { chart } = this;
  const { height, width } = chart;

  const isTop = eventPosition.y > height / 2;
  const yPadding = height / 5;
  const y = isTop ? yPadding : height - yPadding;
  const x = width / 2;

  return { x, y, xAlign: "center", yAlign: "center" };
};

export const commonOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    datalabels: { display: false },
    tooltip: {
      backgroundColor: "#000000cc",
      boxPadding: 5,
      usePointStyle: false,
      borderWidth: 0.00001,
      mode: "index",
      intersect: false,
    },
  },
};

export function tooltipLabelColor(useBorder = false) {
  return function (item) {
    const { backgroundColor, borderColor } = item.element.options;
    const color = useBorder ? borderColor : backgroundColor;
    return {
      borderColor: !item.raw ? colors.muted : "#fff",
      borderWidth: 1,
      backgroundColor: color,
    };
  };
}
