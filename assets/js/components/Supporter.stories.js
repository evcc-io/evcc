import Supporter from "./Supporter.vue";

export default {
  title: "Main/Footer/Supporter",
  component: Supporter,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { Supporter },
  template: '<Supporter v-bind="$props"></Supporter>',
});

export const Standard = Template.bind({});
Standard.args = {
  supporter: false,
};

export const IsSupporter = Template.bind({});
IsSupporter.args = {
  supporter: true,
};
