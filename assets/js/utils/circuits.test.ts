import { describe, expect, test } from "vite-plus/test";
import { circuitTree } from "./circuits";

describe("circuitTree", () => {
  test("single root", () => {
    const result = circuitTree([{ name: "main", power: 0 }]);
    expect(result).toEqual({ name: "main", power: 0 });
  });

  test("root with children", () => {
    const result = circuitTree([
      { name: "root", power: 0 },
      { name: "child1", power: 0, parent: "root" },
      { name: "child2", power: 0, parent: "root" },
    ]);
    expect(result).toEqual({
      name: "root",
      power: 0,
      children: [
        { name: "child1", power: 0, parent: "root" },
        { name: "child2", power: 0, parent: "root" },
      ],
    });
  });

  test("nested two levels", () => {
    const result = circuitTree([
      { name: "root", power: 0 },
      { name: "mid", power: 0, parent: "root" },
      { name: "leaf", power: 0, parent: "mid" },
    ]);
    expect(result).toEqual({
      name: "root",
      power: 0,
      children: [
        {
          name: "mid",
          power: 0,
          parent: "root",
          children: [{ name: "leaf", power: 0, parent: "mid" }],
        },
      ],
    });
  });

  test("empty input", () => {
    expect(circuitTree([])).toBeNull();
  });
});
