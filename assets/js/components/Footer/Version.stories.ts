import Version from "./Version.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Footer/Version",
  component: Version,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    installed: { control: "text" },
    available: { control: "text" },
    commit: { control: "text" },
  },
} as Meta<typeof Version>;

const Template: StoryFn<typeof Version> = (args) => ({
  components: { Version },
  setup() {
    return { args };
  },
  template: '<Version v-bind="args" />',
});

export const Latest = Template.bind({});
Latest.args = {
  installed: "0.303.1",
  available: "0.303.1",
};

export const Nightly = Template.bind({});
Nightly.args = {
  installed: "0.303.1",
  available: "0.303.1",
  commit: "5ce7be4",
};

export const UpdateAvailable = Template.bind({});
UpdateAvailable.args = {
  installed: "0.303.0",
  available: "0.303.1",
};

export const NoReleaseNotes = Template.bind({});
NoReleaseNotes.args = {
  installed: "0.303.0",
  available: "0.303.1",
};
