import Loadpoint from "./Loadpoint.vue";

export default {
  title: "Main/Loadpoint",
  component: Loadpoint,
  argTypes: {
    mode: { control: { type: "inline-radio", options: ["off", "now", "minpv", "pv"] } },
    remoteDisabled: { control: { type: "radio", options: ["", "soft", "hard"] } },
    climater: { control: { type: "inline-radio", options: ["on", "heating", "cooling"] } },
  },
};

const Template = (args) => ({
  setup() {
    return { args };
  },
  components: { Loadpoint },
  template: '<Loadpoint v-bind="args"></Loadpoint>',
});

export const Base = Template.bind({});
Base.args = {
  id: 0,
  pvConfigured: true,
  chargePower: 2800,
  chargedEnergy: 11e3,
  chargeDuration: 95 * 60,
  vehiclePresent: true,
  vehicleTitle: "Mein Auto",
  enabled: true,
  connected: true,
  mode: "pv",
  charging: true,
  vehicleSoC: 66,
  targetSoC: 90,
  chargeCurrent: 7,
  minCurrent: 6,
  maxCurrent: 16,
  activePhases: 2,
};

export const WithoutSoc = Template.bind({});
WithoutSoc.args = {
  id: 0,
  pvConfigured: true,
  chargePower: 2800,
  chargedEnergy: 7123,
  chargeDuration: 95 * 60,
  vehiclePresent: false,
  enabled: true,
  connected: true,
  mode: "pv",
  charging: true,
  vehicleSoC: 66,
  targetSoC: 90,
  chargeCurrent: 7,
  minCurrent: 6,
  maxCurrent: 16,
  activePhases: 2,
};

export const Idle = Template.bind({});
Idle.args = {
  id: 0,
  chargePower: 0,
  vehiclePresent: false,
  enabled: false,
  connected: false,
  mode: "off",
  charging: false,
  chargeCurrent: 0,
  minCurrent: 6,
  maxCurrent: 16,
  activePhases: 0,
};

export const Disabled = Template.bind({});
Disabled.args = {
  id: 0,
  pvConfigured: true,
  remoteDisabled: "soft",
  remoteDisabledSource: "Sunny Home Manager",
  vehiclePresent: true,
  vehicleTitle: "Mein Auto",
  chargedEnergy: 31211,
  enabled: true,
  mode: "now",
  connected: true,
  charging: false,
  chargePower: 8112,
  vehicleSoC: 66,
  targetSoC: 100,
  chargeCurrent: 7,
  minCurrent: 6,
  maxCurrent: 16,
  activePhases: 3,
};
