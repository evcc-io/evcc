import Mode from "./Mode.vue";

export default {
  title: "Main/Mode",
  component: Mode,
  argTypes: {
    mode: { control: { type: "inline-radio", options: ["off", "now", "minpv", "pv"] } },
  },
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { Mode },
  template: '<Mode v-bind="$props"></Mode>',
});

export const Base = Template.bind({});
Base.args = {};

export const CaptionAndPV = Template.bind({});
CaptionAndPV.args = {
  caption: true,
  pvConfigured: true,
  mode: "pv",
};
