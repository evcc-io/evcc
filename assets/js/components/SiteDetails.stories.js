import SiteDetails from "./SiteDetails.vue";

export default {
  title: "Main/SiteDetails",
  component: SiteDetails,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
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
