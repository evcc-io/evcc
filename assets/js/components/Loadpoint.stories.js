import Loadpoint from "./Loadpoint.vue";

export default {
  title: "Main/Loadpoint",
  component: Loadpoint,
  argTypes: {
    mode: { control: { type: "inline-radio", options: ["off", "now", "minpv", "pv"] } },
    climater: { control: { type: "inline-radio", options: ["on", "heating", "cooling"] } },
  },
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { Loadpoint },
  template: '<Loadpoint v-bind="$props"></Loadpoint>',
});

export const Base = Template.bind({});
Base.args = {
  id: 0,
  pvConfigured: true,
};

export const WithLevels = Template.bind({});
WithLevels.args = {
  id: 0,
  pvConfigured: true,
  socLevels: [20, 50, 80, 100],
};
