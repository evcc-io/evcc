import SiteDetails from "./SiteDetails.vue";
import i18n from "../i18n";

export default {
  title: "Main/SiteDetails",
  component: SiteDetails,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  i18n,
  props: Object.keys(argTypes),
  components: { SiteDetails },
  template: '<SiteDetails v-bind="$props"></SiteDetails>',
});

export const GridAndPV = Template.bind({});
GridAndPV.args = {
  gridConfigured: true,
  pvConfigured: true,
  gridPower: 100,
  pvPower: 100,
};

export const WithBattery = Template.bind({});
WithBattery.args = {
  gridConfigured: true,
  pvConfigured: true,
  batteryConfigured: true,
  gridPower: 100,
  pvPower: 100,
  batteryPower: 100,
  batterySoC: 0,
};
