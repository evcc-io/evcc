import { shallowMount } from "@vue/test-utils";
import { describe, expect, it } from "vite-plus/test";
import BatteryConfigCard from "./BatteryConfigCard.vue";
import { createI18n } from "vue-i18n";
import en from "../../../../i18n/en.json";

describe("PV control availability", () => {
  it("explains PV-only starts and hides the conflicting SoC start selector", () => {
    const wrapper = shallowMount(BatteryConfigCard, {
      props: { coordinatedPv: true, bufferSoc: 50, bufferStartSoc: 50 },
      global: {
        renderStubDefaultSlot: true,
        plugins: [createI18n({ legacy: false, locale: "en", messages: { en } })],
        stubs: { "i18n-t": false },
      },
    });
    expect(wrapper.text()).toContain("Battery SoC alone does not start charging");
    expect(wrapper.find("#batteryBufferStart").exists()).toBe(false);
    expect(wrapper.find("#batteryBuffer").exists()).toBe(true);
    wrapper.unmount();
  });
  it("explains blocked control while preserving the configured threshold", () => {
    const wrapper = shallowMount(BatteryConfigCard, {
      props: { batteryPvControlUnavailable: true, batteryPvStartPower: 750 },
      global: {
        renderStubDefaultSlot: true,
        mocks: { $t: (key: string) => key, $i18n: { locale: "de-DE" } },
      },
    });
    expect(wrapper.get('[role="status"]').text()).toBe("battery.config.pvControlUnavailable");
    expect((wrapper.get("#batteryPVStartPower").element as HTMLInputElement).value).toBe("750");
    wrapper.unmount();
  });
});
