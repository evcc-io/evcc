import Vehicle from "./Vehicle.vue";

export default {
  title: "Main/Vehicle",
  component: Vehicle,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { Vehicle },
  template: '<Vehicle v-bind="$props"></Vehicle>',
});

export const Base = Template.bind({});
Base.args = {
  soc: false,
  socTitle: "Mein Auto",
  connected: true,
};

export const VehicleConfigured = Template.bind({});
VehicleConfigured.args = {
  soc: true,
  socTitle: "Mein Auto",
  connected: true,
  socCharge: 15,
};

export const VehicleConfiguredWithMin = Template.bind({});
VehicleConfiguredWithMin.args = {
  soc: true,
  socTitle: "Mein Auto",
  connected: true,
  socCharge: 15,
  minSoC: 30,
};
