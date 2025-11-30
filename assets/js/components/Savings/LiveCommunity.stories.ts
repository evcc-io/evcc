import LiveCommunity from "./LiveCommunity.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Savings/LiveCommunity",
  component: LiveCommunity,
  parameters: {
    layout: "centered",
  },
} as Meta<typeof LiveCommunity>;

const Template: StoryFn<typeof LiveCommunity> = () => ({
  components: { LiveCommunity },
  template: "<LiveCommunity />",
});

export const Default = Template.bind({});
