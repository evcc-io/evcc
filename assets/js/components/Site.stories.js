import Site from "./Site.vue";

export default {
  title: "Main/Site",
  component: Site,
  argTypes: {},
};

const Template = (args) => ({
  setup() {
    return { args };
  },
  components: { Site },
  template: '<Site v-bind="args"></Site>',
});

export const Base = Template.bind({});
Base.args = {
  gridConfigured: true,
  pvConfigured: true,
  gridPower: 1340,
  pvPower: 6420,
  batteryConfigured: true,
  batteryPower: 800,
  batterySoC: 66,
  loadpoints: [],
};

export const Single = Template.bind({});
Single.args = {
  gridConfigured: true,
  pvConfigured: true,
  gridPower: -2300,
  pvPower: 7000,
  batteryConfigured: true,
  batteryPower: -2000,
  batterySoC: 24,
  loadpoints: [
    {
      title: "Ladepunkt 1",
      mode: "now",
      vehicleTitle: "Mein Auto",
      enabled: true,
      connected: true,
      vehiclePresent: true,
      charging: true,
      vehicleSoC: 66,
      vehicleRange: 344,
      targetSoC: 90,
      chargeRemainingDuration: 999,
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
  gridPower: 4000,
  pvPower: 11500,
  batteryConfigured: false,
  batteryPower: 0,
  batterySoC: 0,
  loadpoints: [
    {
      title: "Ladepunkt 1",
      mode: "now",
      vehicleTitle: "Mein Auto",
      enabled: true,
      connected: true,
      vehiclePresent: true,
      charging: true,
      vehicleSoC: 66,
      vehicleRange: 344,
      targetSoC: 90,
      chargeRemainingDuration: 999,
      chargePower: 11232,
      chargeDuration: 123982,
      chargedEnergy: 23213,
    },
    {
      title: "Ladepunkt 2",
      mode: "pv",
      vehicleTitle: "Mein anderes Auto",
    },
    {
      title: "Ladepunkt 3",
      mode: "now",
      vehicleTitle: "Mein drittes Auto",
      enabled: true,
      connected: true,
      charging: true,
      vehicleSoC: 22,
      vehicleRange: 34,
      targetSoC: 100,
      chargeRemainingDuration: 22,
      chargePower: 1423,
      chargeDuration: 243,
      chargedEnergy: 235,
    },
  ],
};
