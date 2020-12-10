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

export const CaptionAndPV = Template.bind({});
CaptionAndPV.args = { caption: true, pv: true, mode: "pv" };

export const Minimal = Template.bind({});
Minimal.args = {};
