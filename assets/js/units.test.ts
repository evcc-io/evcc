import { describe, it, expect } from "vite-plus/test";
import { weekStart } from "./units";

describe("weekStart", () => {
  it("follows the site country", () => {
    expect(weekStart("DE")).toBe(1);
    expect(weekStart("US")).toBe(0);
  });
});
