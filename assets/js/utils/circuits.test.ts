import { describe, expect, test } from "vitest";
import { circuitTree } from "./circuits";

describe("circuitTree", () => {
  test("single root", () => {
    const result = circuitTree({
      main: { power: 0 },
    });
    expect(result).toEqual({ name: "main", power: 0 });
  });

  test("root with children", () => {
    const result = circuitTree({
      root: { power: 0 },
      child1: { power: 0, parent: "root" },
      child2: { power: 0, parent: "root" },
    });
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
    const result = circuitTree({
      root: { power: 0 },
      mid: { power: 0, parent: "root" },
      leaf: { power: 0, parent: "mid" },
    });
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
    expect(circuitTree({})).toBeNull();
  });
});
