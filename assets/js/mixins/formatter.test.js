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
