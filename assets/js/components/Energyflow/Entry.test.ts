import { mount } from "@vue/test-utils";
import { beforeAll, describe, expect, test } from "vite-plus/test";

let Entry: typeof import("./Entry.vue").default;

beforeAll(async () => {
  Object.defineProperty(window, "localStorage", { value: {}, configurable: true });
  Entry = (await import("./Entry.vue")).default;
});

describe("energyflow details", () => {
  const hasDetails = (details: number | null) => {
    const wrapper = mount(Entry, {
      props: { details, detailsFmt: (value: number) => String(value) },
    });
    return wrapper
      .find('[data-testid="energyflow-entry-details"]')
      .findComponent({
        name: "AnimatedNumber",
      })
      .exists();
  };

  test("hides an unknown price", () => {
    expect(hasDetails(null)).toBe(false);
  });

  test("shows a real zero price", () => {
    expect(hasDetails(0)).toBe(true);
  });
});
