import { describe, expect, test } from "vite-plus/test";
import { circuitTree, limitBar } from "./circuits";

describe("circuitTree", () => {
  test("single root", () => {
    const result = circuitTree({
      "db:1": { name: "main", power: 0 },
    });
    expect(result).toEqual({ name: "main", power: 0 });
  });

  test("root with children", () => {
    const result = circuitTree({
      "db:1": { name: "main", power: 0 },
      "db:2": { name: "circuit2", power: 0, parent: "db:1" },
      "db:3": { name: "circuit3", power: 0, parent: "db:1" },
    });
    expect(result).toEqual({
      name: "main",
      power: 0,
      children: [
        { name: "circuit2", power: 0, parent: "db:1" },
        { name: "circuit3", power: 0, parent: "db:1" },
      ],
    });
  });

  test("nested two levels", () => {
    const result = circuitTree({
      "db:1": { name: "main", power: 0 },
      "db:2": { name: "circuit2", power: 0, parent: "db:1" },
      "db:3": { name: "circuit3", power: 0, parent: "db:2" },
    });
    expect(result).toEqual({
      name: "main",
      power: 0,
      children: [
        {
          name: "circuit2",
          power: 0,
          parent: "db:1",
          children: [{ name: "circuit3", power: 0, parent: "db:2" }],
        },
      ],
    });
  });

  test("empty input", () => {
    expect(circuitTree({})).toBeUndefined();
  });
});

describe("limitBar", () => {
  test("below limit", () => {
    expect(limitBar(5, 10)).toEqual({ state: "normal", fill: 50 });
  });

  test("near limit", () => {
    expect(limitBar(9.5, 10)).toEqual({ state: "warning", fill: 100 });
  });

  test("over limit rescales to value", () => {
    const bar = limitBar(15, 10);
    expect(bar.state).toBe("danger");
    expect(bar.fill).toBeCloseTo(66.67, 1);
  });

  test("without limit", () => {
    expect(limitBar(5)).toEqual({ state: "normal", fill: 100 });
  });
});
