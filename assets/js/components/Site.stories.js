import Site from "./Site.vue";
import i18n from "../i18n";

export default {
  title: "Main/Site",
  component: Site,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  i18n,
  props: Object.keys(argTypes),
  components: { Site },
  template: '<Site v-bind="$props"></Site>',
});

export const Base = Template.bind({});
Base.args = {
  gridConfigured: true,
  pvConfigured: true,
  gridPower: 100,
  pvPower: 100,
  batteryConfigured: true,
  batteryPower: 100,
  batterySoC: 0,
  loadpoints: [],
};

export const Single = Template.bind({});
Single.args = {
  gridConfigured: true,
  pvConfigured: true,
  gridPower: 100,
  pvPower: 100,
  batteryConfigured: true,
  batteryPower: 100,
  batterySoC: 0,
  loadpoints: [
    {
      title: "Ladepunkt 1",
      mode: "now",
      socTitle: "Mein Auto",
      enabled: true,
      connected: true,
      hasVehicle: true,
      charging: true,
      socCharge: 66,
      targetSoC: 90,
      range: 344,
      chargeEstimate: 999,
      chargePower: 11232,
      chargeDuration: 123982,
      chargedEnergy: 23213,
    },
  ],
};

export const Multi = Template.bind({});
Multi.args = {
  gridConfigured: true,
  pvConfigured: true,
  gridPower: -100,
  pvPower: 100,
  batteryConfigured: true,
  batteryPower: 100,
  batterySoC: 60,
  loadpoints: [
    {
      title: "Ladepunkt 1",
      mode: "now",
      socTitle: "Mein Auto",
      enabled: true,
      connected: true,
      hasVehicle: true,
      charging: true,
      socCharge: 66,
      targetSoC: 90,
      range: 344,
      chargeEstimate: 999,
      chargePower: 11232,
      chargeDuration: 123982,
      chargedEnergy: 23213,
    },
    {
      title: "Ladepunkt 2",
      mode: "pv",
    },
  ],
};
