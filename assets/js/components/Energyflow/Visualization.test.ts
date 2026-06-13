import { mount, config } from "@vue/test-utils";
import { describe, expect, test } from "vitest";
import Visualization from "./Visualization.vue";
import { POWER_UNIT } from "@/mixins/formatter";

config.global.mocks["$t"] = (key: string) => key;

const mountBar = (props: object) =>
  mount(Visualization, { props: { powerUnit: POWER_UNIT.KW, ...props } });

describe("small segment visibility", () => {
  test("small self-consumption keeps a visible width and minimum bar width", () => {
    // grid dominates, self-pv is below the former 2% threshold (80 / 5080 ≈ 1.6%)
    const wrapper = mountBar({ gridImport: 5000, selfPv: 80 });
    const selfPv = wrapper.get(".self-pv");

    // segment is rendered proportionally, not dropped to zero width …
    expect(selfPv.attributes("style")).toContain("width:");
    expect(selfPv.attributes("style")).not.toContain("width: 0;");
    // … and clamped to a minimum so the green color stays visible
    expect(selfPv.attributes("style")).toContain("min-width:");
  });

  test("zero segment has no width and no minimum width", () => {
    const wrapper = mountBar({ gridImport: 5000, selfPv: 0 });
    const selfPv = wrapper.get(".self-pv");

    expect(selfPv.attributes("style")).toContain("width: 0");
    expect(selfPv.attributes("style") ?? "").not.toContain("min-width:");
  });
});
