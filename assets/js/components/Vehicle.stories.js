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

export const Car = Template.bind({});
Car.args = {
  soc: true,
  socTitle: "Mein Auto",
  connected: true,
  socCharge: 15,
};

export const CarWithMin = Template.bind({});
CarWithMin.args = {
  soc: true,
  socTitle: "Mein Auto",
  connected: true,
  socCharge: 15,
  minSoC: 30,
};
