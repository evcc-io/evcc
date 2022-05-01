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
  template: '<div class="d-flex justify-content-end"><Savings v-bind="args"></Savings></div>',
});

export const Default = Template.bind({});
Default.args = {
  since: 82800,
  totalCharged: 15231,
  selfConsumptionCharged: 12231,
  selfConsumptionPercent: 80.3,
};

export const NoCharge = Template.bind({});
NoCharge.args = {
  since: 82800,
  totalCharged: 0,
};
