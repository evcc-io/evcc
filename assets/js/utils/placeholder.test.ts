import { describe, it, expect } from "vitest";
import { extractPlaceholders as extract, replacePlaceholders as replace } from "./placeholder";

describe("placeholder", () => {
  it("extract", () => {
    expect(extract("homes")).toEqual([]);
    expect(extract("")).toEqual([]);
    expect(extract("a/{home}/b")).toEqual(["home"]);
    expect(extract("{a}/{b}")).toEqual(["a", "b"]);
    expect(extract("{p1}/{p2}")).toEqual(["p1", "p2"]);
    expect(extract("{a_b}/{c_d}")).toEqual(["a_b", "c_d"]);
    expect(extract("{a}/{a}")).toEqual(["a", "a"]);
  });

  it("replace", () => {
    expect(replace("homes", {})).toBe("homes");
    expect(replace("a/{b}/c", { b: "x" })).toBe("a/x/c");
    expect(replace("{a}/{b}", { a: "1", b: "2" })).toBe("1/2");
    expect(replace("{a}", { a: "b c" })).toBe("b%20c");
    expect(replace("{a}", { a: "b/c" })).toBe("b%2Fc");
    expect(replace("{a}", { a: "b+c" })).toBe("b%2Bc");
    expect(replace("{a}", {})).toBe("{a}");
    expect(replace("{a}", { a: "" })).toBe("");
    expect(replace("{a}/{b}", { a: "x" })).toBe("x/{b}");
  });
});
