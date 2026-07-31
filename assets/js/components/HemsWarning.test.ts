import { mount, config } from "@vue/test-utils";
import { describe, expect, test } from "vite-plus/test";
import HemsWarning from "./HemsWarning.vue";
import type { HemsStatus } from "@/types/evcc";
import en from "../../../i18n/en.json";

// minimal $t that walks en.json so tests assert on real English text
const lookup = (key: string): string | undefined => {
  const v = key.split(".").reduce<any>((o, k) => o?.[k], en);
  return typeof v === "string" ? v : undefined;
};
config.global.mocks["$t"] = (key: string) => lookup(key) ?? key;
config.global.mocks["$i18n"] = { locale: "de-DE" };

const banner = (status: HemsStatus, gridPower: number) => {
  const wrapper = mount(HemsWarning, { props: { status, gridPower } });
  return wrapper.find("[data-testid=hems-warning]");
};

const DIMMED: HemsStatus = { dimmed: true, maxConsumptionPower: 4200 };
const CURTAILED: HemsStatus = { curtailed: 60, maxProductionPower: 6000 };

describe("visibility", () => {
  test("hidden without limits", () => {
    expect(banner({ dimmed: false, curtailed: 100 }, 4000).exists()).eq(false);
  });

  test("hidden while the limit is not approached", () => {
    expect(banner(DIMMED, 2939).exists()).eq(false); // 69%
    expect(banner(CURTAILED, -4199).exists()).eq(false); // 69%
  });

  test("visible from 70% of the limit", () => {
    expect(banner(DIMMED, 2940).text()).contains("Consumption");
    expect(banner(CURTAILED, -4200).text()).contains("Feed-in");
  });

  test("only the approached limit is shown", () => {
    const both = { ...DIMMED, ...CURTAILED };
    expect(banner(both, 4000).text()).not.contains("Feed-in");
    expect(banner(both, -6000).text()).not.contains("Consumption");
  });

  test("a zero limit is exhausted by any power in its direction", () => {
    const full: HemsStatus = { curtailed: 0, maxProductionPower: 0 };
    expect(banner(full, 0).exists()).eq(false);
    expect(banner(full, -1).text()).contains("0,0 kW");
  });
});

describe("severity", () => {
  test("warning below 90% of the limit", () => {
    expect(banner(DIMMED, 3779).classes()).not.contains("limit-stripe--critical");
  });

  test("critical from 90% of the limit", () => {
    expect(banner(DIMMED, 3780).classes()).contains("limit-stripe--critical");
    expect(banner(CURTAILED, -5400).classes()).contains("limit-stripe--critical");
  });
});
