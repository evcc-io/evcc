import { describe, expect, test } from "vite-plus/test";
import { sinkCost, sourceBreakdown } from "./mix";

describe("sourceBreakdown", () => {
  test("scales the sink's split and prices to the entity", () => {
    const rows = sourceBreakdown(
      [
        { from: "pv", to: "loadpoint", energy: 6, cost: 0.3, pricedEnergy: 3 },
        { from: "grid", to: "loadpoint", energy: 2, cost: 0.6, pricedEnergy: 2 },
        { from: "grid", to: "home", energy: 9 },
      ],
      "loadpoint",
      4,
      ["pv", "grid"]
    );
    expect(rows).toEqual([
      { from: "pv", share: 75, energy: 3, price: expect.closeTo(0.1), cost: expect.closeTo(0.3) },
      { from: "grid", share: 25, energy: 1, price: expect.closeTo(0.3), cost: expect.closeTo(0.3) },
    ]);
  });

  test("no cost without priced energy, only the given sources", () => {
    const rows = sourceBreakdown([{ from: "pv", to: "home", energy: 1 }], "home", 1, [
      "pv",
      "battery",
    ]);
    expect(rows).toEqual([
      { from: "pv", share: 100, energy: 1, price: undefined, cost: undefined },
      { from: "battery", share: 0, energy: 0, price: undefined, cost: undefined },
    ]);
  });
});

describe("sinkCost", () => {
  test("sums cost and priced energy of the flows into the sinks", () => {
    const flows = [
      { from: "pv", to: "home", energy: 2, cost: 0.2, pricedEnergy: 2 },
      { from: "grid", to: "loadpoint", energy: 3, cost: 0.6, pricedEnergy: 2 },
      { from: "pv", to: "export", energy: 5 },
    ] as const;
    expect(sinkCost([...flows], "home")).toEqual({ cost: 0.2, energy: 2 });
    expect(sinkCost([...flows], "home", "loadpoint")).toEqual({
      cost: expect.closeTo(0.8),
      energy: 4,
    });
    expect(sinkCost([...flows], "export")).toEqual({ cost: 0, energy: 0 });
  });
});
