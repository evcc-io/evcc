import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, test, vi } from "vite-plus/test";
import type { UiLoadpoint } from "@/types/evcc";

const { post } = vi.hoisted(() => {
  Object.defineProperty(window, "localStorage", { value: {}, configurable: true });
  return { post: vi.fn() };
});

vi.mock("@/api", () => ({ default: { post } }));
vi.mock("@/mixins/formatter", () => ({
  default: {
    methods: {
      fmtNumber: (value: number) => String(value),
      fmtPercentage: (value: number) => `${value}%`,
      fmtTemperature: (value: number) => `${value} °C`,
      fmtPhasePower: () => "0 kW",
      fmtCurrent: (value: number) => `${value} A`,
    },
  },
}));

import SettingsModal from "./SettingsModal.vue";

describe("dehumidifier settings", () => {
  beforeEach(() => post.mockClear());

  test("posts target humidity and minimum-on time", async () => {
    const wrapper = mount(SettingsModal, {
      props: {
        loadpoints: [
          {
            id: "1",
            chargerFeatureDehumidifier: true,
            dehumidifier: {
              targetHumidity: 55,
              hysteresis: 2,
              minOnTime: 600_000_000_000,
              minOffTime: 300_000_000_000,
            },
          } as UiLoadpoint,
        ],
      },
      global: {
        mocks: { $t: (key: string) => key },
        stubs: {
          GenericModal: { template: "<div><slot /></div>", methods: { open() {}, close() {} } },
          SmartCostLimit: true,
          SmartFeedInPriority: true,
          LoadpointSettingsBatteryBoost: true,
        },
      },
    });

    wrapper.vm.open("1");
    await wrapper.vm.$nextTick();
    await wrapper.get("#loadpoint_1_humiditytarget").setValue("62.5");
    await wrapper.get("#loadpoint_1_humiditytarget").trigger("change");
    await wrapper.get("#loadpoint_1_dehumidifierminontime").setValue("12");
    await wrapper.get("#loadpoint_1_dehumidifierminontime").trigger("change");

    expect(post).toHaveBeenCalledWith("loadpoints/1/dehumidifier/target/62.5");
    expect(post).toHaveBeenCalledWith("loadpoints/1/dehumidifier/minontime/720s");
  });
});
