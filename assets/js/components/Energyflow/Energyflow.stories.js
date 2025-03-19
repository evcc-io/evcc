import Energyflow from "./Energyflow.vue";

export default {
  title: "Energyflow/Energyflow",
  component: Energyflow,
};

const Template = (args) => ({
  components: { Energyflow },
  setup() {
    return { args };
  },
  template: '<Energyflow v-bind="args" />',
});

export const GridAndPV = Template.bind({});
GridAndPV.args = {
  gridConfigured: true,
  pvConfigured: true,
  pvPower: 7300,
  gridPower: -2300,
  homePower: 800,
  loadpointsCompact: [
    { power: 1000, icon: "car", charging: true },
    { power: 1000, icon: "bike", charging: true },
    { power: 2200, icon: "car", charging: true },
  ],
  tariffGrid: 0.25,
  tariffFeedIn: 0.08,
  tariffEffectivePrice: 0.08,
  smartCostType: "price",
  smartCostAvailable: true,
  currency: "EUR",
  siteTitle: "Home",
  pv: [{ power: 5000 }, { power: 2300 }],
};

export const BatteryAndGrid = Template.bind({});
BatteryAndGrid.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 0,
  gridPower: 1200,
  homePower: 2000,
  batteryPower: 800,
  batterySoc: 77,
  tariffGrid: 0.25,
  tariffFeedIn: 0.08,
  tariffEffectivePrice: 0.08,
  currency: "EUR",
  siteTitle: "Home",
  battery: [
    { soc: 44.999, capacity: 13.3, power: 350 },
    { soc: 82.3331, capacity: 21, power: 450 },
  ],
};

export const BatteryCharging = Template.bind({});
BatteryCharging.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 5000,
  gridPower: -1300,
  homePower: 800,
  loadpointsCompact: [{ power: 1400, icon: "car", charging: true }],
  batteryPower: -1500,
  batterySoc: 75,
  siteTitle: "Home",
};

export const GridPVAndBattery = Template.bind({});
GridPVAndBattery.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 1000,
  gridPower: 700,
  homePower: 3300,
  batteryPower: 1500,
  batterySoc: 30,
  siteTitle: "Home",
};

export const BatteryThresholds = Template.bind({});
BatteryThresholds.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 8700,
  gridPower: -500,
  loadpointsCompact: [
    { power: 5000, icon: "car", charging: true },
    { power: 2500, icon: "bus", charging: true },
  ],
  batteryPower: -700,
  batterySoc: 95,
  siteTitle: "Home",
};

export const PVThresholds = Template.bind({});
PVThresholds.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 300,
  gridPower: 6500,
  homePower: 1000,
  loadpointsCompact: [
    { power: 5000, icon: "car", charging: true },
    { power: 1600, icon: "car", charging: true },
  ],
  batteryPower: 800,
  batterySoc: 76,
  siteTitle: "Home",
};

export const GridOnly = Template.bind({});
GridOnly.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 0,
  gridPower: 6500,
  homePower: 1000,
  loadpointsCompact: [
    { power: 5500, icon: "car", charging: true },
    { power: 0, icon: "car", charging: false },
    { power: 0, icon: "car", charging: false },
    { power: 0, icon: "car", charging: false },
  ],
  batteryPower: 0,
  batterySoc: 0,
  siteTitle: "Home",
};

export const LowPower = Template.bind({});
LowPower.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 700,
  gridPower: -300,
  homePower: 300,
  batteryPower: -100,
  batterySoc: 55,
  tariffGrid: 0.25,
  tariffFeedIn: 0.08,
  tariffEffectivePrice: 0.08,
  currency: "EUR",
  siteTitle: "Home",
};

export const CO2 = Template.bind({});
CO2.args = {
  gridConfigured: true,
  pvConfigured: true,
  pvPower: 7300,
  gridPower: -2300,
  homePower: 800,
  loadpointsCompact: [
    { power: 1000, icon: "car", charging: true },
    { power: 1000, icon: "car", charging: true },
    { power: 2200, icon: "car", charging: true },
  ],
  tariffGrid: 0.25,
  tariffFeedIn: 0.08,
  tariffEffectivePrice: 0.08,
  tariffCo2: 723,
  tariffEffectiveCo2: 0,
  smartCostType: "co2",
  smartCostAvailable: true,
  currency: "EUR",
  siteTitle: "Home",
  pv: [{ power: 5000 }, { power: 2300 }],
};

export const UnknownInput = Template.bind({});
UnknownInput.args = {
  gridConfigured: true,
  pvConfigured: true,
  pvPower: 2000,
  gridPower: -2000,
  loadpointsCompact: [{ power: 1000, icon: "car", charging: true }],
};

export const UnknownInputFill = Template.bind({});
UnknownInputFill.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 500,
  gridPower: 0,
  batteryPower: -1000,
  loadpointsCompact: [],
};

export const UnknownOutput = Template.bind({});
UnknownOutput.args = {
  gridConfigured: true,
  pvConfigured: true,
  pvPower: 3000,
  gridPower: -1000,
  loadpointsCompact: [{ power: 1700, icon: "car", charging: true }],
};

export const UnknownOutputLessThan10Percent = Template.bind({});
UnknownOutputLessThan10Percent.args = {
  gridConfigured: true,
  pvConfigured: true,
  pvPower: 3000,
  gridPower: -1000,
  loadpointsCompact: [{ power: 1800, icon: "car", charging: true }],
};
