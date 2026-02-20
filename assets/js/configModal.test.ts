import { describe, expect, test } from "vitest";
import { parseKey, parseQueryString, buildQuery, extractQueryString } from "./configModal";

describe("parseKey", () => {
  test("parses bracket notation", () => {
    expect(parseKey("meter")).toEqual({ name: "meter" });
    expect(parseKey("meter[type:grid]")).toEqual({ name: "meter", type: "grid" });
    expect(parseKey("meter[choices:pv,battery]")).toEqual({
      name: "meter",
      choices: ["pv", "battery"],
    });
  });
});

describe("parseQueryString", () => {
  test("parses basic queries", () => {
    expect(parseQueryString("")).toEqual([]);
    expect(parseQueryString("messaging")).toEqual([{ name: "messaging" }]);
    expect(parseQueryString("meter=1")).toEqual([{ name: "meter", id: 1 }]);
    expect(parseQueryString("messaging=")).toEqual([{ name: "messaging" }]);
  });

  test("parses multiple modals", () => {
    expect(parseQueryString("meter=1&vehicle=2")).toEqual([
      { name: "meter", id: 1 },
      { name: "vehicle", id: 2 },
    ]);
  });

  test("parses type and choices", () => {
    expect(parseQueryString("meter[type:grid]=1")).toEqual([
      { name: "meter", type: "grid", id: 1 },
    ]);
    expect(parseQueryString("meter[choices:pv,battery]")).toEqual([
      { name: "meter", choices: ["pv", "battery"] },
    ]);
  });

  test("handles edge cases", () => {
    expect(parseQueryString("meter=1&callbackCompleted=true")).toEqual([{ name: "meter", id: 1 }]);
    expect(parseQueryString("meter%5Btype%3Agrid%5D=1")).toEqual([
      { name: "meter", type: "grid", id: 1 },
    ]);
    expect(parseQueryString("meter=abc")).toEqual([{ name: "meter" }]);
  });
});

describe("buildQuery", () => {
  test("builds query objects", () => {
    expect(buildQuery([])).toEqual({});
    expect(buildQuery([{ name: "messaging" }])).toEqual({ messaging: "" });
    expect(buildQuery([{ name: "meter", id: 1 }])).toEqual({ meter: "1" });
    expect(buildQuery([{ name: "meter", type: "grid", id: 1 }])).toEqual({
      "meter[type:grid]": "1",
    });
    expect(buildQuery([{ name: "meter", choices: ["pv", "battery"] }])).toEqual({
      "meter[choices:pv,battery]": "",
    });
    expect(
      buildQuery([
        { name: "meter", id: 1 },
        { name: "vehicle", id: 2 },
      ])
    ).toEqual({ meter: "1", vehicle: "2" });
  });
});

describe("extractQueryString", () => {
  test("extracts query from paths", () => {
    expect(extractQueryString("/config?meter=1")).toBe("meter=1");
    expect(extractQueryString("/config")).toBe("");
    expect(extractQueryString("/#/config?meter=1")).toBe("meter=1");
    expect(extractQueryString("/config?meter=1&vehicle=2")).toBe("meter=1&vehicle=2");
  });
});
