import { describe, expect, test } from "vite-plus/test";
import { tooltipTable } from "./echarts";

describe("tooltipTable", () => {
  test("escapes head, headers and names", () => {
    const html = tooltipTable("<img a>", [{ name: "<img b>", values: ["1"] }], ["<img c>"]);
    expect(html).not.toContain("<img");
    expect(html).toContain("&lt;img a&gt;");
    expect(html).toContain("&lt;img b&gt;");
    expect(html).toContain("&lt;img c&gt;");
  });
});
