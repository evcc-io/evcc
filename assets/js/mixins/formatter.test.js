import { mount, config } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import formatter, { WATT_FORMAT } from "./formatter";

config.global.mocks["$i18n"] = { locale: "de-DE" };
config.global.mocks["$t"] = (a) => a;

const fmt = mount({
  render() {},
  mixins: [formatter],
}).componentVM;

describe("fmtW", () => {
  test("should format with units", () => {
    expect(fmt.fmtW(0, WATT_FORMAT.AUTO)).eq("0 W");
    expect(fmt.fmtW(1200000, WATT_FORMAT.AUTO)).eq("1,2 mW");
    expect(fmt.fmtW(0, WATT_FORMAT.MW)).eq("0,0 mW");
    expect(fmt.fmtW(1200000, WATT_FORMAT.MW)).eq("1,2 mW");
    expect(fmt.fmtW(0, WATT_FORMAT.KW)).eq("0,0 kW");
    expect(fmt.fmtW(1200000, WATT_FORMAT.KW)).eq("1.200,0 kW");
    expect(fmt.fmtW(0, WATT_FORMAT.WATT)).eq("0 W");
    expect(fmt.fmtW(1200000, WATT_FORMAT.WATT)).eq("1.200.000 W");
  });
  test("should format without units", () => {
    expect(fmt.fmtW(0, WATT_FORMAT.AUTO, false)).eq("0");
    expect(fmt.fmtW(1200000, WATT_FORMAT.AUTO, false)).eq("1,2");
    expect(fmt.fmtW(0, WATT_FORMAT.MW, false)).eq("0,0");
    expect(fmt.fmtW(1200000, WATT_FORMAT.MW, false)).eq("1,2");
    expect(fmt.fmtW(0, WATT_FORMAT.KW, false)).eq("0,0");
    expect(fmt.fmtW(1200000, WATT_FORMAT.KW, false)).eq("1.200,0");
    expect(fmt.fmtW(0, WATT_FORMAT.WATT, false)).eq("0");
    expect(fmt.fmtW(1200000, WATT_FORMAT.WATT, false)).eq("1.200.000");
  });
  test("should format a given number of digits", () => {
    expect(fmt.fmtW(12345, WATT_FORMAT.AUTO, true, 0)).eq("12 kW");
    expect(fmt.fmtW(12345, WATT_FORMAT.AUTO, true, 1)).eq("12,3 kW");
    expect(fmt.fmtW(12345, WATT_FORMAT.AUTO, true, 2)).eq("12,35 kW");
    expect(fmt.fmtW(12345, WATT_FORMAT.MW, true, 0)).eq("0 mW");
    expect(fmt.fmtW(12345, WATT_FORMAT.MW, true, 1)).eq("0,0 mW");
    expect(fmt.fmtW(12345, WATT_FORMAT.MW, true, 2)).eq("0,01 mW");
    expect(fmt.fmtW(12345, WATT_FORMAT.KW, true, 0)).eq("12 kW");
    expect(fmt.fmtW(12345, WATT_FORMAT.KW, true, 1)).eq("12,3 kW");
    expect(fmt.fmtW(12345, WATT_FORMAT.KW, true, 2)).eq("12,35 kW");
    expect(fmt.fmtW(12345, WATT_FORMAT.WATT, true, 0)).eq("12.345 kW");
    expect(fmt.fmtW(12345, WATT_FORMAT.WATT, true, 1)).eq("12.345,0 kW");
    expect(fmt.fmtW(12345, WATT_FORMAT.WATT, true, 2)).eq("12.345,00 kW");
  });
});

describe("fmtWh", () => {
  test("should format with units", () => {
    expect(fmt.fmtWh(0, WATT_FORMAT.AUTO)).eq("0 Wh");
    expect(fmt.fmtWh(1200000, WATT_FORMAT.AUTO)).eq("1,2 mWh");
    expect(fmt.fmtWh(0, WATT_FORMAT.MW)).eq("0,0 mWh");
    expect(fmt.fmtWh(1200000, WATT_FORMAT.MW)).eq("1,2 mWh");
    expect(fmt.fmtWh(0, WATT_FORMAT.KW)).eq("0,0 kWh");
    expect(fmt.fmtWh(1200000, WATT_FORMAT.KW)).eq("1.200,0 kWh");
    expect(fmt.fmtWh(0, WATT_FORMAT.WATT)).eq("0 Wh");
    expect(fmt.fmtWh(1200000, WATT_FORMAT.WATT)).eq("1.200.000 Wh");
  });
  test("should format without units", () => {
    expect(fmt.fmtW(0, WATT_FORMAT.AUTO, false)).eq("0");
    expect(fmt.fmtW(1200000, WATT_FORMAT.AUTO, false)).eq("1,2");
    expect(fmt.fmtWh(0, WATT_FORMAT.MW, false)).eq("0,0");
    expect(fmt.fmtWh(1200000, WATT_FORMAT.MW, false)).eq("1,2");
    expect(fmt.fmtWh(0, WATT_FORMAT.KW, false)).eq("0,0");
    expect(fmt.fmtWh(1200000, WATT_FORMAT.KW, false)).eq("1.200,0");
    expect(fmt.fmtWh(0, WATT_FORMAT.WATT, false)).eq("0");
    expect(fmt.fmtWh(1200000, WATT_FORMAT.WATT, false)).eq("1.200.000");
  });
  test("should format a given number of digits", () => {
    expect(fmt.fmtWh(12345, WATT_FORMAT.AUTO, true, 0)).eq("12 kWh");
    expect(fmt.fmtWh(12345, WATT_FORMAT.AUTO, true, 1)).eq("12,3 kWh");
    expect(fmt.fmtWh(12345, WATT_FORMAT.AUTO, true, 2)).eq("12,35 kWh");
    expect(fmt.fmtWh(12345, WATT_FORMAT.MW, true, 0)).eq("0 mWh");
    expect(fmt.fmtWh(12345, WATT_FORMAT.MW, true, 1)).eq("0,0 mWh");
    expect(fmt.fmtWh(12345, WATT_FORMAT.MW, true, 2)).eq("0,01 mWh");
    expect(fmt.fmtWh(12345, WATT_FORMAT.KW, true, 0)).eq("12 kWh");
    expect(fmt.fmtWh(12345, WATT_FORMAT.KW, true, 1)).eq("12,3 kWh");
    expect(fmt.fmtWh(12345, WATT_FORMAT.KW, true, 2)).eq("12,35 kWh");
    expect(fmt.fmtWh(12345, WATT_FORMAT.WATT, true, 0)).eq("12.345 kWh");
    expect(fmt.fmtWh(12345, WATT_FORMAT.WATT, true, 1)).eq("12.345,0 kWh");
    expect(fmt.fmtWh(12345, WATT_FORMAT.WATT, true, 2)).eq("12.345,00 kWh");
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
