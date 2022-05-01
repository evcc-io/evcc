import Mode from "./Mode.vue";

export default {
  title: "Main/Mode",
  component: Mode,
  argTypes: {
    mode: { control: { type: "inline-radio" }, options: ["off", "now", "minpv", "pv"] },
  },
};

const Template = (args) => ({
  setup() {
    return { args };
  },
  components: { Mode },
  template: '<Mode v-bind="args"></Mode>',
});

export const Base = Template.bind({});
Base.args = { mode: "pv" };
