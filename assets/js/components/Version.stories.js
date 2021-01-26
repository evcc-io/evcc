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
};
export const ReleaseNotes = Template.bind({});
ReleaseNotes.args = {
  installed: "0.36",
  available: "0.40",
  releaseNotes: "Lorem ipsum dolor sit amet consectetur",
};

export const Updater = Template.bind({});
Updater.args = {
  installed: "0.36",
  available: "0.40",
  releaseNotes: "Lorem ipsum dolor sit amet consectetur",
  hasUpdater: true,
};

export const Upgrade = Template.bind({});
Upgrade.args = {
  installed: "0.36",
  available: "0.40",
  hasUpdater: true,
};
