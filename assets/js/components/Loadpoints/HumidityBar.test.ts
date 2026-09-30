import { mount, config } from "@vue/test-utils";
import { describe, expect, test } from "vite-plus/test";
import HumidityBar from "./HumidityBar.vue";
import en from "../../../../i18n/en.json";

const lookup = (key: string): string | undefined => {
  const value = key.split(".").reduce<any>((object, part) => object?.[part], en);
  return typeof value === "string" ? value : undefined;
};

config.global.mocks["$t"] = (key: string) => lookup(key) ?? key;
config.global.mocks["$i18n"] = { locale: "en-US" };

describe("humidity bar", () => {
  test("shows measured humidity and target marker", () => {
    const wrapper = mount(HumidityBar, {
      props: { humidity: 63.4, targetHumidity: 55, enabled: true },
    });

    expect(wrapper.get('[data-testid="humidity-fill"]').attributes("aria-valuenow")).toBe("63.4");
    expect(wrapper.get('[data-testid="humidity-fill"]').element.getAttribute("style")).toContain(
      "width: 63.4%"
    );
    expect(
      wrapper.get('[data-testid="humidity-target-marker"]').element.getAttribute("style")
    ).toContain("left: 55%");
    expect(wrapper.get('[data-testid="humidity-current-value"]').text()).toBe("63.4 %RH");
    expect(wrapper.get('[data-testid="humidity-target-value"]').text()).toBe("55.0 %RH");
    expect(wrapper.get('[data-testid="humidity-device-state"]').text()).toContain("On");
  });

  test("does not render a false reading when humidity is unavailable", () => {
    const wrapper = mount(HumidityBar, { props: { humidity: null, targetHumidity: 55 } });

    expect(wrapper.findAll('[role="progressbar"]').length).toBe(0);
    expect(wrapper.get('[data-testid="humidity-current-value"]').text()).toBe("Unavailable");
    expect(wrapper.findAll('[data-testid="humidity-target-marker"]').length).toBe(1);
  });
});
