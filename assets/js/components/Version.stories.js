import Version from "./Version.vue";

export default {
  title: "Main/Version",
  component: Version,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { Version },
  template: '<Version v-bind="$props"></Version>',
});

export const Base = Template.bind({});
Base.args = {
  installed: "0.36",
  available: "0.40",
  releaseNotes: "<li>feature 1<li>feature 2",
};
