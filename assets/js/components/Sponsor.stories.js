import Sponsor from "./Sponsor.vue";
import i18n from "../i18n";

export default {
  title: "Main/Footer/Sponsor",
  component: Sponsor,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  i18n,
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
