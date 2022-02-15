import Savings from "./Savings.vue";

export default {
  title: "Main/Footer/Savings",
  component: Savings,
  argTypes: {},
};

const Template = (args) => ({
  setup() {
    return { args };
  },
  components: { Savings },
  template: '<Savings v-bind="args"></Savings>',
});

export const Default = Template.bind({});
Default.args = {};

export const Default2 = Template.bind({});
Default2.args = {};
