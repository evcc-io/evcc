import Sponsor from "./Sponsor.vue";

export default {
  title: "Main/Footer/Sponsor",
  component: Sponsor,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { Sponsor },
  template: '<Sponsor v-bind="$props"></Sponsor>',
});

export const Standard = Template.bind({});
Standard.args = {
  sponsor: undefined,
};

export const IsSponsor = Template.bind({});
IsSponsor.args = {
  sponsor: "naltatis",
};
