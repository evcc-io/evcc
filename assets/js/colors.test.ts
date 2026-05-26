import { describe, it, expect } from "vitest";
import colors, { dimColor, lighterColor, fullColor, resolveColors, deviceColorMap } from "./colors";
import type { DeviceColors } from "./types/evcc";

describe("setAlpha helpers", () => {
  it("appends alpha to 6-digit hex", () => {
    expect(dimColor("#abcdef")).toBe("#abcdef20");
    expect(lighterColor("#abcdef")).toBe("#abcdefaa");
    expect(fullColor("#abcdef")).toBe("#abcdefff");
  });

  it("replaces alpha on 8-digit hex", () => {
    expect(dimColor("#abcdefff")).toBe("#abcdef20");
    expect(lighterColor("#abcdef12")).toBe("#abcdefaa");
    expect(fullColor("#abcdef00")).toBe("#abcdefff");
  });

  it("handles null/undefined", () => {
    expect(dimColor(null)).toBeUndefined();
    expect(lighterColor(null)).toBeUndefined();
    expect(fullColor(null)).toBeUndefined();
  });

  it("returns input unchanged for unexpected length", () => {
    expect(dimColor("rgb(1,2,3)")).toBe("rgb(1,2,3)");
  });
});

describe("resolveColors", () => {
  const palette = colors.palette;

  it("returns empty map for empty ids", () => {
    expect(resolveColors([], {})).toEqual({});
  });

  it("autoassigns palette in order without overrides", () => {
    const ids = ["a", "b", "c"];
    expect(resolveColors(ids, {})).toEqual({
      a: palette[0],
      b: palette[1],
      c: palette[2],
    });
  });

  it("respects explicit override", () => {
    const res = resolveColors(["a", "b"], { a: "#123456" });
    expect(res["a"]).toBe("#123456");
    expect(res["b"]).toBe(palette[0]);
  });

  it("skips overridden palette entry on autoassign", () => {
    // override "a" with first palette color → others should not reuse it
    const res = resolveColors(["a", "b", "c"], { a: palette[0] });
    expect(res["a"]).toBe(palette[0]);
    expect(res["b"]).toBe(palette[1]);
    expect(res["c"]).toBe(palette[2]);
  });

  it("empty-string override falls back to autoassign", () => {
    const res = resolveColors(["a", "b"], { a: "" });
    expect(res["a"]).toBe(palette[0]);
    expect(res["b"]).toBe(palette[1]);
  });

  it("normalizes case when matching taken colors", () => {
    // override uses lowercase; should still skip the matching palette entry
    const lower = palette[0].toLowerCase();
    const res = resolveColors(["a", "b"], { a: lower });
    expect(res["a"]).toBe(lower);
    expect(res["b"]).toBe(palette[1]);
  });

  it("wraps around once palette exhausted", () => {
    const ids = Array.from({ length: palette.length + 2 }, (_, i) => `id${i}`);
    const res = resolveColors(ids, {});
    expect(res[ids[0]]).toBe(palette[0]);
    expect(res[ids[palette.length]]).toBe(palette[0]);
    expect(res[ids[palette.length + 1]]).toBe(palette[1]);
  });

  it("with every palette entry overridden, autoassign wraps palette", () => {
    const overrides: DeviceColors = {};
    palette.forEach((c, i) => (overrides[`o${i}`] = c));
    const ids = [...Object.keys(overrides), "extra"];
    const res = resolveColors(ids, overrides);
    palette.forEach((c, i) => expect(res[`o${i}`]).toBe(c));
    // free list empty → falls back to wrap-around on palette
    expect(res["extra"]).toBe(palette[0]);
  });
});

describe("deviceColorMap", () => {
  it("returns empty map for undefined", () => {
    expect(deviceColorMap(undefined)).toEqual({});
  });
  it("returns empty map for empty list", () => {
    expect(deviceColorMap([])).toEqual({});
  });
  it("converts list of entries to title→color map", () => {
    expect(
      deviceColorMap([
        { title: "WP-SG+", color: "#2563EB" },
        { title: "Heizung", color: "#DC2626" },
      ])
    ).toEqual({ "WP-SG+": "#2563EB", Heizung: "#DC2626" });
  });
});
