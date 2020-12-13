import Soc from "./Soc.vue";

export default {
  title: "Main/Soc",
  component: Soc,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { Soc },
  template: '<Soc v-bind="$props"></Soc>',
});

export const Base = Template.bind({});
Base.args = {
  soc: 80,
  levels: [20, 50, 80, 100],
};

export const Caption = Template.bind({});
Caption.args = {
  caption: true,
  soc: 80,
  levels: [20, 50, 80, 100],
};
