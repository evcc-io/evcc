import "popper.js";
import "bootstrap";
import Mode from "../assets/js/components/Mode.vue";

export default {
  title: "Example/Mode",
  component: Mode,
  argTypes: {
    pv: "pv",
    mode: { control: { type: "select", options: ["off", "now", "minpv", "pv"] } },
  },
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { Mode },
  template: '<div><Mode v-bind="$props"></Mode></div>',
});

export const Primary = Template.bind({});
Primary.args = {};
