import Mode from "./Mode.vue";
import i18n from "../i18n";

export default {
  title: "Main/Mode",
  component: Mode,
  argTypes: {
    mode: { control: { type: "inline-radio", options: ["off", "now", "minpv", "pv"] } },
  },
};

const Template = (args, { argTypes }) => ({
  i18n,
  props: Object.keys(argTypes),
  components: { Mode },
  template: '<Mode v-bind="$props"></Mode>',
});

export const Base = Template.bind({});
Base.args = { mode: "pv" };
