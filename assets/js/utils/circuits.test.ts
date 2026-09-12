import { describe, expect, test } from "vite-plus/test";
import { circuitTree } from "./circuits";

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
