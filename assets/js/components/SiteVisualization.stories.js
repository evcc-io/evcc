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
  loadpoints: [{ chargePower: 4500 }],
};

export const WithBattery = Template.bind({});
WithBattery.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  gridPower: 2400,
  pvPower: 800,
  batteryPower: 100,
  batterySoC: 0,
  loadpoints: [{ chargePower: 1800 }],
};
