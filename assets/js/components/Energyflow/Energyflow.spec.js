/* globals describe, it, expect */
import { shallowMount } from "@vue/test-utils";
import Energyflow from "./Energyflow.vue";

const mocks = {
  $t: (x) => x,
  $tc: (x, n) => `${n} ${x}`,
  $n: function (value, options) {
    const n = new Intl.NumberFormat("en-EN", options);
    return n.format(value);
  },
};

describe("Energyflow.vue", () => {
  const defaultProps = {
    gridConfigured: true,
    gridPower: 0,
    pvConfigured: true,
    pvPower: 0,
    homePower: 0,
    batteryConfigured: false,
    batteryPower: 0,
    batterySoC: 0,
  };

  it("using pv and grid power", async () => {
    const wrapper = shallowMount(Energyflow, {
      mocks,
      propsData: {
        ...defaultProps,
        gridPower: 1000,
        pvPower: 4000,
        homePower: 1300,
        loadpointsPower: 3700,
      },
    });
    await wrapper.find(".energyflow").trigger("click");

    expect(wrapper.find("[data-test-grid-import]").text()).toMatch("1.0 kW");
    expect(wrapper.find("[data-test-self-consumption]").text()).toMatch("4.0 kW");
    expect(wrapper.find("[data-test-pv-export]").text()).toMatch("0.0 kW");

    expect(wrapper.find("[data-test-home-power]").text()).toMatch("1.3 kW");
    expect(wrapper.find("[data-test-loadpoints]").text()).toMatch("3.7 kW");
    expect(wrapper.find("[data-test-pv-production]").text()).toMatch("4.0 kW");
    expect(wrapper.find("[data-test-battery]").exists()).toBe(false);
  });

  it("exporting all pv power, no usage", async () => {
    const wrapper = shallowMount(Energyflow, {
      mocks,
      propsData: { ...defaultProps, gridPower: -4000, pvPower: 5000, loadpointsPower: 1000 },
    });

    await wrapper.find(".energyflow").trigger("click");

    expect(wrapper.find("[data-test-grid-import]").text()).toMatch("0.0 kW");
    expect(wrapper.find("[data-test-self-consumption]").text()).toMatch("1.0 kW");
    expect(wrapper.find("[data-test-pv-export]").text()).toMatch("4.0 kW");

    expect(wrapper.find("[data-test-home-power]").text()).toMatch("0.0 kW");
    expect(wrapper.find("[data-test-loadpoints]").text()).toMatch("1.0 kW");
    expect(wrapper.find("[data-test-pv-production]").text()).toMatch("5.0 kW");
    expect(wrapper.find("[data-test-battery]").exists()).toBe(false);
  });

  it("more grid export than pv, grid value wins (invalid state)", async () => {
    const wrapper = shallowMount(Energyflow, {
      mocks,
      propsData: { ...defaultProps, gridPower: -4000, pvPower: 3000 },
    });

    await wrapper.find(".energyflow").trigger("click");

    expect(wrapper.find("[data-test-grid-import]").text()).toMatch("0.0 kW");
    expect(wrapper.find("[data-test-self-consumption]").text()).toMatch("0.0 kW");
    expect(wrapper.find("[data-test-pv-export]").text()).toMatch("4.0 kW");

    expect(wrapper.find("[data-test-home-power]").text()).toMatch("0.0 kW");
    expect(wrapper.find("[data-test-loadpoints]").text()).toMatch("0.0 kW");
    expect(wrapper.find("[data-test-pv-production]").text()).toMatch("3.0 kW");
    expect(wrapper.find("[data-test-battery]").exists()).toBe(false);
  });

  it("more loadpoints power than grid import, loadpoints unchanged, house consumption min 0 (invalid state)", async () => {
    const wrapper = shallowMount(Energyflow, {
      mocks,
      propsData: { ...defaultProps, gridPower: 6000, pvPower: 0, loadpointsPower: 7000 },
    });

    await wrapper.find(".energyflow").trigger("click");

    expect(wrapper.find("[data-test-grid-import]").text()).toMatch("6.0 kW");
    expect(wrapper.find("[data-test-self-consumption]").text()).toMatch("0.0 kW");
    expect(wrapper.find("[data-test-pv-export]").text()).toMatch("0.0 kW");

    expect(wrapper.find("[data-test-home-power]").text()).toMatch("0.0 kW");
    expect(wrapper.find("[data-test-loadpoints]").text()).toMatch("7.0 kW");
    expect(wrapper.find("[data-test-pv-production]").text()).toMatch("0.0 kW");
    expect(wrapper.find("[data-test-battery]").exists()).toBe(false);
  });

  it("only grid usage, no pv, idleBattery", async () => {
    const wrapper = shallowMount(Energyflow, {
      mocks,
      propsData: {
        ...defaultProps,
        gridPower: 360,
        pvPower: 0,
        batteryConfigured: true,
        batteryPower: 0,
        homePower: 360,
      },
    });

    await wrapper.find(".energyflow").trigger("click");

    expect(wrapper.find("[data-test-grid-import]").text()).toMatch("360 W");
    expect(wrapper.find("[data-test-self-consumption]").text()).toMatch("0 W");
    expect(wrapper.find("[data-test-pv-export]").text()).toMatch("0 W");

    expect(wrapper.find("[data-test-home-power]").text()).toMatch("360 W");
    expect(wrapper.find("[data-test-pv-production]").text()).toMatch("0 W");
    expect(wrapper.find("[data-test-battery]").text()).toMatch("main.energyflow.battery");
  });

  it("grid and battery usage, no pv", async () => {
    const wrapper = shallowMount(Energyflow, {
      mocks,
      propsData: {
        ...defaultProps,
        gridPower: 300,
        batteryConfigured: true,
        batteryPower: 234,
        batterySoC: 77,
        pvPower: 0,
        homePower: 534,
      },
    });

    await wrapper.find(".energyflow").trigger("click");

    expect(wrapper.find("[data-test-grid-import]").text()).toMatch("300 W");
    expect(wrapper.find("[data-test-self-consumption]").text()).toMatch("234 W");
    expect(wrapper.find("[data-test-pv-export]").text()).toMatch("0 W");

    expect(wrapper.find("[data-test-home-power]").text()).toMatch("534 W");
    expect(wrapper.find("[data-test-pv-production]").text()).toMatch("0 W");
    expect(wrapper.find("[data-test-battery]").text()).toMatch("234 W");
    expect(wrapper.find("[data-test-battery]").text()).toMatch("77%");
    expect(wrapper.find("[data-test-battery]").text()).toMatch("main.energyflow.batteryDischarge");
  });

  it("battery charge, pv export", async () => {
    const wrapper = shallowMount(Energyflow, {
      mocks,
      propsData: {
        ...defaultProps,
        gridPower: -2500,
        batteryConfigured: true,
        batteryPower: -1700,
        pvPower: 9000,
        homePower: 4800,
      },
    });

    await wrapper.find(".energyflow").trigger("click");

    expect(wrapper.find("[data-test-grid-import]").text()).toMatch("0.0 kW");
    expect(wrapper.find("[data-test-self-consumption]").text()).toMatch("6.5 kW");
    expect(wrapper.find("[data-test-pv-export]").text()).toMatch("2.5 kW");

    expect(wrapper.find("[data-test-home-power]").text()).toMatch("4.8 kW");
    expect(wrapper.find("[data-test-pv-production]").text()).toMatch("9.0 kW");
    expect(wrapper.find("[data-test-battery]").text()).toMatch("1.7 kW");
    expect(wrapper.find("[data-test-battery]").text()).toMatch("main.energyflow.batteryCharge");
  });

  it("thresholds", async () => {
    const wrapper = shallowMount(Energyflow, {
      mocks,
      propsData: {
        ...defaultProps,
        gridPower: 5555,
        batteryConfigured: true,
        batteryPower: 1234,
        pvPower: 378,
        homePower: 7200,
      },
    });

    await wrapper.find(".energyflow").trigger("click");

    expect(wrapper.find("[data-test-grid-import]").text()).toMatch("5.6 kW");
    expect(wrapper.find("[data-test-self-consumption]").text()).toMatch("1.6 kW");
    expect(wrapper.find("[data-test-pv-export]").text()).toMatch("0.0 kW");

    expect(wrapper.find("[data-test-home-power]").text()).toMatch("7.2 kW");
    expect(wrapper.find("[data-test-pv-production]").text()).toMatch("0.4 kW");
    expect(wrapper.find("[data-test-battery]").text()).toMatch("1.2 kW");
    expect(wrapper.find("[data-test-battery]").text()).toMatch("main.energyflow.batteryDischarge");
  });

  it("battery charge, grid import", async () => {
    const wrapper = shallowMount(Energyflow, {
      mocks,
      propsData: {
        ...defaultProps,
        gridPower: 1500,
        batteryConfigured: true,
        batteryPower: -1000,
        homePower: 500,
        pvPower: 0,
      },
    });

    await wrapper.find(".energyflow").trigger("click");

    expect(wrapper.find("[data-test-grid-import]").text()).toMatch("1.5 kW");
    expect(wrapper.find("[data-test-self-consumption]").text()).toMatch("0.0 kW");
    expect(wrapper.find("[data-test-pv-export]").text()).toMatch("0.0 kW");

    expect(wrapper.find("[data-test-home-power]").text()).toMatch("0.5 kW");
    expect(wrapper.find("[data-test-pv-production]").text()).toMatch("0.0 kW");
    expect(wrapper.find("[data-test-battery]").text()).toMatch("1.0 kW");
    expect(wrapper.find("[data-test-battery]").text()).toMatch("main.energyflow.batteryCharge");
  });
});
