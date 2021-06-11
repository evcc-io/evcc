import SiteVisualization from "./SiteVisualization.vue";
import i18n from "../i18n";

export default {
  title: "Main/SiteVisualization",
  component: SiteVisualization,
  argTypes: {
    gridPower: { control: { type: "range", min: -5000, max: 20000, step: 100 } },
    pvPower: { control: { type: "range", min: 0, max: 10000, step: 100 } },
    batteryPower: { control: { type: "range", min: -4000, max: 4000, step: 100 } },
  },
};

const Template = (args, { argTypes }) => ({
  i18n,
  props: Object.keys(argTypes),
  components: { SiteVisualization },
  template: '<SiteVisualization v-bind="$props"></SiteVisualization>',
});

export const GridAndPV = Template.bind({});
GridAndPV.args = {
  gridConfigured: true,
  pvConfigured: true,
  gridPower: -2300,
  pvPower: 7320,
  batteryConfigured: false,
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
};

export const BatteryCharging = Template.bind({});
BatteryCharging.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  gridPower: -1300,
  pvPower: 5000,
  batteryPower: -1500,
  batterySoC: 49,
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
};

export const SmallPowerThresholds = Template.bind({});
SmallPowerThresholds.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  gridPower: -110,
  pvPower: 8740,
  batteryPower: -100,
  batterySoC: 30,
};

export const GridOnly = Template.bind({});
GridOnly.args = {
  gridConfigured: true,
  pvConfigured: false,
  batteryConfigured: false,
  gridPower: -6230,
  pvPower: 0,
  batteryPower: 0,
  batterySoC: 0,
};
