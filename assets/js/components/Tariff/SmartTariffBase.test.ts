import { mount, config } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import SmartTariffBase from "./SmartTariffBase.vue";

config.global.mocks["$t"] = (key: string) => key;
config.global.mocks["$i18n"] = { locale: "en-US" };

const createWrapper = () =>
  mount(SmartTariffBase, {
    props: {
      currentLimit: null,
      lastLimit: 0,
      possible: true,
      formId: "smart-cost",
      limitLabel: "Limit",
      activeHoursLabel: "Active hours",
      currentPriceLabel: "Current price",
      isSlotActive: () => false,
      tariff: [],
    },
    global: {
      stubs: {
        TariffChart: { template: "<div />" },
      },
    },
  });

describe("SmartTariffBase", () => {
  test("toggles chart scale buttons", async () => {
    const wrapper = createWrapper();

    const zeroButton = wrapper
      .findAll("button")
      .find((button) => button.text() === "smartCost.chartScaleZero");
    const rangeButton = wrapper
      .findAll("button")
      .find((button) => button.text() === "smartCost.chartScaleRange");

    expect(zeroButton).toBeDefined();
    expect(rangeButton).toBeDefined();

    expect(rangeButton?.classes()).toContain("active");
    expect(zeroButton?.classes()).not.toContain("active");

    await zeroButton?.trigger("click");

    expect(zeroButton?.classes()).toContain("active");
    expect(rangeButton?.classes()).not.toContain("active");
  });
});
