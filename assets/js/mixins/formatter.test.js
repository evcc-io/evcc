import { mount, config } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import formatter from "./formatter";

config.global.mocks["$i18n"] = { locale: "de-DE" };
config.global.mocks["$t"] = (a) => a;

const fmt = mount({
  render() {},
  mixins: [formatter],
}).componentVM;

describe("fmtKWh", () => {
  test("should format with units", () => {
    expect(fmt.fmtKWh(1200)).eq("1,2 kWh");
    expect(fmt.fmtKWh(1200, true)).eq("1,2 kWh");
    expect(fmt.fmtKWh(1200, false)).eq("1.200 Wh");
    expect(fmt.fmtKWh(1200, false, false)).eq("1.200");
  });
  test("should format with digits", () => {
    expect(fmt.fmtKWh(56789)).eq("56,8 kWh");
    expect(fmt.fmtKWh(56789, true, true, 0)).eq("57 kWh");
    expect(fmt.fmtKWh(56789, true, true, 1)).eq("56,8 kWh");
    expect(fmt.fmtKWh(56789, true, true, 2)).eq("56,79 kWh");
    expect(fmt.fmtKWh(56789.123, false, true)).eq("56.789 Wh");
    expect(fmt.fmtKWh(56789.123, false, true, 2)).eq("56.789,12 Wh");
  });
});

describe("fmtPricePerKWh", () => {
  test("should format with units", () => {
    expect(fmt.fmtPricePerKWh(0.2, "EUR")).eq("20,0 ct/kWh");
    expect(fmt.fmtPricePerKWh(0.2, "EUR", true)).eq("20,0 ct");
    expect(fmt.fmtPricePerKWh(0.234, "USD")).eq("23,4 ct/kWh");
    expect(fmt.fmtPricePerKWh(1234, "SEK")).eq("1.234,0 SEK/kWh");
    expect(fmt.fmtPricePerKWh(0.2, "EUR", false, false)).eq("20,0");
  });
});

describe("pricePerKWhUnit", () => {
  test("should return correct unit", () => {
    expect(fmt.pricePerKWhUnit("EUR")).eq("ct/kWh");
    expect(fmt.pricePerKWhUnit("EUR", true)).eq("ct");
    expect(fmt.pricePerKWhUnit("USD")).eq("ct/kWh");
    expect(fmt.pricePerKWhUnit("SEK")).eq("SEK/kWh");
    expect(fmt.pricePerKWhUnit("SEK", true)).eq("SEK");
  });
});
