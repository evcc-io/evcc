import Sponsor from "./Sponsor.vue";

export default {
  title: "Main/Footer/Sponsor",
  component: Sponsor,
  argTypes: {},
};

const Template = (args) => ({
  setup() {
    return { args };
  },
  components: { Sponsor },
  template: '<Sponsor v-bind="args"></Sponsor>',
});

export const Standard = Template.bind({});
Standard.args = {
  sponsor: undefined,
};

export const IsSponsor = Template.bind({});
IsSponsor.args = {
  sponsor: "naltatis",
};
