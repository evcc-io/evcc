import { mount } from "@vue/test-utils";
import { beforeAll, describe, expect, test } from "vite-plus/test";

let SessionInfo: typeof import("./SessionInfo.vue").default;

beforeAll(async () => {
  Object.defineProperty(window, "localStorage", { value: {}, configurable: true });
  SessionInfo = (await import("./SessionInfo.vue")).default;
});

describe("session price options", () => {
  const options = (tariffPriceLoadpoints: number | null) => {
    const wrapper = mount(SessionInfo, {
      props: {
        tariffPriceLoadpoints,
        sessionPrice: null,
        sessionPricePerKWh: null,
      },
      global: {
        mocks: {
          $t: (key: string) => key,
          $i18n: { locale: "en" },
        },
      },
    });
    return wrapper.vm.optionKeys;
  };

  test("available for an idle session with a charge price", () => {
    expect(options(0.3)).toContain("avgPrice");
    expect(options(0.3)).toContain("price");
  });

  test("available for a zero charge price", () => {
    expect(options(0)).toContain("avgPrice");
    expect(options(0)).toContain("price");
  });

  test("hidden when the charge price is unknown", () => {
    expect(options(null)).not.toContain("avgPrice");
    expect(options(null)).not.toContain("price");
  });
});
