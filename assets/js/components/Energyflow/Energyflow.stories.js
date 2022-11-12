import Energyflow from "./";

export default {
  title: "Main/Energyflow",
  component: Energyflow,
  argTypes: {
    pvPower: { control: { type: "range", min: 0, max: 10000, step: 100 } },
    gridPower: { control: { type: "range", min: -5000, max: 20000, step: 100 } },
    homePower: { control: { type: "range", min: 0, max: 30000, step: 100 } },
    loadpointsPower: { control: { type: "range", min: 0, max: 20000, step: 100 } },
    activeLoadpointsCount: { control: { type: "range", min: 0, max: 5, step: 1 } },
    batteryPower: { control: { type: "range", min: -4000, max: 4000, step: 100 } },
    batterySoC: { control: { type: "range", min: 0, max: 100, step: 1 } },
  },
};

const Template = (args) => ({
  setup() {
    return { args };
  },
  components: { Energyflow },
  template: '<Energyflow v-bind="args"></Energyflow>',
});

export const GridAndPV = Template.bind({});
GridAndPV.args = {
  gridConfigured: true,
  pvConfigured: true,
  pvPower: 7300,
  gridPower: -2300,
  homePower: 800,
  loadpointsPower: 4200,
  activeLoadpointsCount: 3,
  batteryConfigured: false,
  siteTitle: "Home",
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
  batterySoC: 77,
  siteTitle: "Home",
};

export const BatteryCharging = Template.bind({});
BatteryCharging.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 5000,
  gridPower: -1300,
  homePower: 800,
  loadpointsPower: 1400,
  activeLoadpointsCount: 1,
  batteryPower: -1500,
  batterySoC: 75,
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
  batterySoC: 30,
  siteTitle: "Home",
};

export const BatteryThresholds = Template.bind({});
BatteryThresholds.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  activeLoadpointsCount: 1,
  pvPower: 8700,
  gridPower: -500,
  loadpointsPower: 7500,
  batteryPower: -700,
  batterySoC: 95,
  siteTitle: "Home",
};

export const PVThresholds = Template.bind({});
PVThresholds.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  activeLoadpointsCount: 2,
  pvPower: 300,
  gridPower: 5500,
  homePower: 1000,
  loadpointsPower: 5600,
  batteryPower: 800,
  batterySoC: 76,
  siteTitle: "Home",
};

export const GridOnly = Template.bind({});
GridOnly.args = {
  gridConfigured: true,
  pvConfigured: false,
  batteryConfigured: false,
  activeLoadpointsCount: 1,
  pvPower: 0,
  gridPower: -6200,
  homePower: 1000,
  loadpointsPower: 4200,
  batteryPower: 0,
  batterySoC: 0,
  siteTitle: "Home",
};

export const LowEnergy = Template.bind({});
LowEnergy.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  pvPower: 700,
  gridPower: -300,
  homePower: 300,
  batteryPower: -100,
  batterySoC: 55,
  siteTitle: "Home",
};
