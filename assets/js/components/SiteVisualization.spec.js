/* globals describe, it, expect */
import { shallowMount } from "@vue/test-utils";
import SiteVisualization from "./SiteVisualization.vue";

describe("SiteVisualization.vue", () => {
  const defaultProps = {
    gridConfigured: true,
    gridPower: 0,
    pvConfigured: true,
    pvPower: 0,
    batteryConfigured: true,
    batteryPower: 0,
    batterySoC: 0,
    loadpoints: [{}],
  };

  it("using pv and grid power", () => {
    const wrapper = shallowMount(SiteVisualization, {
      propsData: { ...defaultProps, gridPower: 1000, pvPower: 4000 },
    });
    expect(wrapper.find(".usage-label").text()).toMatch("5.0 kW");
    expect(wrapper.find(".pv-export").text()).toMatch("0.0 kW");
    expect(wrapper.find(".surplus").text()).toMatch("0.0 kW");
  });

  it("exporting all pv power, no usage", () => {
    const wrapper = shallowMount(SiteVisualization, {
      propsData: { ...defaultProps, gridPower: -4000, pvPower: 4000 },
    });
    expect(wrapper.find(".usage-label").text()).toMatch("0.0 kW");
    expect(wrapper.find(".pv-export").text()).toMatch("4.0 kW");
    expect(wrapper.find(".surplus").text()).toMatch("4.0 kW");
  });

  it("more grid export than pv, grid value wins (invalid state)", () => {
    const wrapper = shallowMount(SiteVisualization, {
      propsData: { ...defaultProps, gridPower: -4000, pvPower: 3000 },
    });
    expect(wrapper.find(".usage-label").text()).toMatch("0.0 kW");
    expect(wrapper.find(".pv-export").text()).toMatch("4.0 kW");
    expect(wrapper.find(".surplus").text()).toMatch("4.0 kW");
  });

  it("only grid usage, no pv", () => {
    const wrapper = shallowMount(SiteVisualization, {
      propsData: { ...defaultProps, gridPower: 360, pvPower: 0 },
    });
    expect(wrapper.find(".usage-label").text()).toMatch("0.4 kW");
    expect(wrapper.find(".pv-export").text()).toMatch("0.0 kW");
    expect(wrapper.find(".surplus").text()).toMatch("0.0 kW");
  });

  it("grid and battery usage, no pv", () => {
    const wrapper = shallowMount(SiteVisualization, {
      propsData: { ...defaultProps, gridPower: 300, batteryPower: 200, pvPower: 0 },
    });
    expect(wrapper.find(".usage-label").text()).toMatch("0.5 kW");
    expect(wrapper.find(".pv-export").text()).toMatch("0.0 kW");
    expect(wrapper.find(".surplus").text()).toMatch("0.0 kW");
  });
});
