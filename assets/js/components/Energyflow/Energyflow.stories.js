import Energyflow from "./";
import i18n from "../../i18n";

export default {
  title: "Main/Energyflow",
  component: Energyflow,
  argTypes: {
    gridPower: { control: { type: "range", min: -5000, max: 20000, step: 100 } },
    pvPower: { control: { type: "range", min: 0, max: 10000, step: 100 } },
    loadpointsPower: { control: { type: "range", min: 0, max: 20000, step: 100 } },
    activeLoadpointsCount: { control: { type: "range", min: 0, max: 5, step: 1 } },
    batteryPower: { control: { type: "range", min: -4000, max: 4000, step: 100 } },
    batterySoC: { control: { type: "range", min: 0, max: 100, step: 1 } },
  },
};

const Template = (args, { argTypes }) => ({
  i18n,
  props: Object.keys(argTypes),
  components: { Energyflow },
  template: '<Energyflow v-bind="$props"></Energyflow>',
});

export const GridAndPV = Template.bind({});
GridAndPV.args = {
  gridConfigured: true,
  pvConfigured: true,
  gridPower: -2300,
  pvPower: 7320,
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
  gridPower: 1200,
  pvPower: 0,
  batteryPower: 800,
  batterySoC: 77,
  siteTitle: "Home",
};

export const BatteryCharging = Template.bind({});
BatteryCharging.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  gridPower: -1300,
  pvPower: 5000,
  loadpointsPower: 1400,
  activeLoadpointsCount: 1,
  batteryPower: -1500,
  batterySoC: 75,
  siteTitle: "Home",
};

export const GridPvAndBattery = Template.bind({});
GridPvAndBattery.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  gridPower: 700,
  pvPower: 1000,
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
  loadpointsPower: 7523,
  gridPower: -510,
  pvPower: 8740,
  batteryPower: -100,
  batterySoC: 95,
  siteTitle: "Home",
};

export const PvThresholds = Template.bind({});
PvThresholds.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  activeLoadpointsCount: 2,
  loadpointsPower: 5621,
  batteryPower: 800,
  gridPower: 5555,
  pvPower: 300,
  batterySoC: 76,
  siteTitle: "Home",
};

export const GridOnly = Template.bind({});
GridOnly.args = {
  gridConfigured: true,
  pvConfigured: false,
  batteryConfigured: false,
  gridPower: -6230,
  activeLoadpointsCount: 1,
  loadpointsPower: 4200,
  pvPower: 0,
  batteryPower: 0,
  batterySoC: 0,
  siteTitle: "Home",
};

export const LowEnergy = Template.bind({});
LowEnergy.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  gridPower: -352,
  pvPower: 710,
  batteryPower: 86,
  batterySoC: 55,
  siteTitle: "Home",
};
