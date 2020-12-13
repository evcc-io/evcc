import Loadpoint from "./Loadpoint.vue";

export default {
  title: "Main/Loadpoint",
  component: Loadpoint,
  argTypes: {
    climater: { control: { type: "select", options: ["on", "heating", "cooling"] } },
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
