import { describe, expect, it, vi } from "vitest";
import { getActiveDataPoints, syncChartTooltip } from "./chartSync";

describe("chartSync", () => {
  it("returns active points only for visible datasets with data", () => {
    const chart = {
      data: { datasets: [{}, {}, {}] },
      getDatasetMeta: vi.fn((datasetIndex: number) => {
        if (datasetIndex === 0) {
          return { hidden: false, data: [{}, {}] };
        }
        if (datasetIndex === 1) {
          return { hidden: true, data: [{}, {}] };
        }
        return { hidden: false, data: [{}] };
      }),
    };

    expect(getActiveDataPoints(chart as never, 1)).toEqual([{ datasetIndex: 0, index: 1 }]);
  });

  it("clears active elements when index is null", () => {
    const setActiveElements = vi.fn();
    const setTooltipElements = vi.fn();
    const update = vi.fn();

    syncChartTooltip(
      {
        setActiveElements,
        tooltip: { setActiveElements: setTooltipElements },
        update,
      } as never,
      null
    );

    expect(setActiveElements).toHaveBeenCalledWith([]);
    expect(setTooltipElements).toHaveBeenCalledWith([], { x: 0, y: 0 });
    expect(update).toHaveBeenCalledWith("none");
  });
});
