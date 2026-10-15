import { describe, it, expect } from "vite-plus/test";
import { timeZoneCountry } from "./timeZoneCountry";

describe("timeZoneCountry", () => {
  it("maps a zone to its country", () => {
    expect(timeZoneCountry("Europe/Berlin")).toBe("DE");
    expect(timeZoneCountry("Europe/Zurich")).toBe("CH");
    expect(timeZoneCountry("America/New_York")).toBe("US");
  });

  it("gives up on zones without a country", () => {
    expect(timeZoneCountry("UTC")).toBeUndefined();
  });
});
