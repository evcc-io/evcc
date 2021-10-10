import Loadpoint from "./Loadpoint.vue";
import i18n from "../i18n";

export default {
  title: "Main/Loadpoint",
  component: Loadpoint,
  argTypes: {
    mode: { control: { type: "inline-radio", options: ["off", "now", "minpv", "pv"] } },
    remoteDisabled: { control: { type: "radio", options: ["", "soft", "hard"] } },
    climater: { control: { type: "inline-radio", options: ["on", "heating", "cooling"] } },
  },
};

const Template = (args, { argTypes }) => ({
  i18n,
  props: Object.keys(argTypes),
  components: { Loadpoint },
  template: '<Loadpoint v-bind="$props"></Loadpoint>',
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
  charging: true,
  vehicleSoC: 66,
  targetSoC: 90,
};

export const Disabled = Template.bind({});
Disabled.args = {
  id: 0,
  pvConfigured: true,
  remoteDisabled: "soft",
  remoteDisabledSource: "Sunny Home Manager",
  vehiclePresent: true,
  vehicleTitle: "Mein Auto",
  enabled: true,
  connected: true,
  charging: false,
  vehicleSoC: 66,
  targetSoC: 100,
};
