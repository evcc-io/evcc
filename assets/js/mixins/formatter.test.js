import { mount, config } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import formatter from "./formatter";

config.global.mocks["$i18n"] = { locale: "de-DE" };
config.global.mocks["$t"] = (a) => a;

const fmt = mount({
  render() {},
  mixins: [formatter],
}).componentVM;

describe("fmtkW", () => {
  test("should format kW and W", () => {
    expect(fmt.fmtKw(0, true)).eq("0,0 kW");
    expect(fmt.fmtKw(1200, true)).eq("1,2 kW");
    expect(fmt.fmtKw(0, false)).eq("0 W");
    expect(fmt.fmtKw(1200, false)).eq("1.200 W");
  });
  test("should format without unit", () => {
    expect(fmt.fmtKw(0, true, false)).eq("0,0");
    expect(fmt.fmtKw(1200, true, false)).eq("1,2");
    expect(fmt.fmtKw(0, false, false)).eq("0");
    expect(fmt.fmtKw(1200, false, false)).eq("1.200");
  });
  test("should format a given number of digits", () => {
    expect(fmt.fmtKw(12345, true, true, 0)).eq("12 kW");
    expect(fmt.fmtKw(12345, true, true, 1)).eq("12,3 kW");
    expect(fmt.fmtKw(12345, true, true, 2)).eq("12,35 kW");
  });
});

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
    expect(fmt.fmtPricePerKWh(0.234, "USD")).eq("23,4 ¢/kWh");
    expect(fmt.fmtPricePerKWh(1234, "SEK")).eq("1.234,0 SEK/kWh");
    expect(fmt.fmtPricePerKWh(0.2, "EUR", false, false)).eq("20,0");
    expect(fmt.fmtPricePerKWh(0.123, "CHF")).eq("12,3 rp/kWh");
  });
});

describe("pricePerKWhUnit", () => {
  test("should return correct unit", () => {
    expect(fmt.pricePerKWhUnit("EUR")).eq("ct/kWh");
    expect(fmt.pricePerKWhUnit("EUR", true)).eq("ct");
    expect(fmt.pricePerKWhUnit("USD")).eq("¢/kWh");
    expect(fmt.pricePerKWhUnit("SEK")).eq("SEK/kWh");
    expect(fmt.pricePerKWhUnit("SEK", true)).eq("SEK");
    expect(fmt.pricePerKWhUnit("CHF")).eq("rp/kWh");
  });
});

describe("fmtDuration", () => {
  test("should format zero duration", () => {
    expect(fmt.fmtDuration(0)).eq("—");
    expect(fmt.fmtDuration(-100)).eq("—");
  });
  test("should format seconds", () => {
    expect(fmt.fmtDuration(1)).eq("1\u202Fs");
    expect(fmt.fmtDuration(59)).eq("59\u202Fs");
    expect(fmt.fmtDuration(59, false)).eq("59");
    expect(fmt.fmtDuration(59, true, "m")).eq("0:59\u202Fm");
    expect(fmt.fmtDuration(59, true, "h")).eq("0:00\u202Fh");
  });
  test("should format minutes", () => {
    expect(fmt.fmtDuration(60)).eq("1:00\u202Fm");
    expect(fmt.fmtDuration(150)).eq("2:30\u202Fm");
    expect(fmt.fmtDuration(150, false)).eq("2:30");
    expect(fmt.fmtDuration(150, true, "h")).eq("0:02\u202Fh");
  });
  test("should format hours", () => {
    expect(fmt.fmtDuration(60 * 60)).eq("1:00\u202Fh");
    expect(fmt.fmtDuration(60 * 60 * 2.5)).eq("2:30\u202Fh");
    expect(fmt.fmtDuration(60 * 60 * 2.5, false)).eq("2:30");
  });
  test("should format internationalized", () => {
    config.global.mocks["$i18n"].locale = "ar-EG";
    expect(fmt.fmtDuration(30, false)).eq("٣٠");
    expect(fmt.fmtDuration(90, false)).eq("١:٣٠");
    expect(fmt.fmtDuration(60 * 100, false)).eq("١:٤٠");
    config.global.mocks["$i18n"].locale = "de-DE";
  });
});
