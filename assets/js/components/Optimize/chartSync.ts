import type { ActiveDataPoint } from "chart.js";

type ChartMetaLike = {
  hidden?: boolean;
  data: unknown[];
};

export interface SyncableChart {
  data: {
    datasets: unknown[];
  };
  getDatasetMeta: (datasetIndex: number) => ChartMetaLike;
  setActiveElements: (active: ActiveDataPoint[]) => void;
  tooltip?: {
    setActiveElements: (active: ActiveDataPoint[], eventPosition: { x: number; y: number }) => void;
  };
  update: (mode?: "none") => void;
}

const getCenterPoint = (element: unknown): { x: number; y: number } | null => {
  if (
    element &&
    typeof element === "object" &&
    "getCenterPoint" in element &&
    typeof element.getCenterPoint === "function"
  ) {
    return element.getCenterPoint();
  }

  return null;
};

export const getActiveDataPoints = (
  chart: Pick<SyncableChart, "data" | "getDatasetMeta">,
  index: number
): ActiveDataPoint[] => {
  return chart.data.datasets.flatMap((_dataset, datasetIndex) => {
    const meta = chart.getDatasetMeta(datasetIndex) as ChartMetaLike;
    if (meta.hidden || !meta.data[index]) {
      return [];
    }
    return [{ datasetIndex, index }];
  });
};

export const syncChartTooltip = (chart: SyncableChart | null | undefined, index: number | null) => {
  if (!chart) {
    return;
  }

  if (index === null) {
    chart.setActiveElements([]);
    chart.tooltip?.setActiveElements([], { x: 0, y: 0 });
    chart.update("none");
    return;
  }

  const activeElements = getActiveDataPoints(chart, index);
  if (!activeElements.length) {
    chart.setActiveElements([]);
    chart.tooltip?.setActiveElements([], { x: 0, y: 0 });
    chart.update("none");
    return;
  }

  const firstMeta = chart.getDatasetMeta(activeElements[0]!.datasetIndex) as ChartMetaLike;
  const firstElement = firstMeta.data[activeElements[0]!.index];
  const position = getCenterPoint(firstElement) ?? { x: 0, y: 0 };

  chart.setActiveElements(activeElements);
  chart.tooltip?.setActiveElements(activeElements, position);
  chart.update("none");
};
