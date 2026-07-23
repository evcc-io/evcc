import SponsorTokenExpires from "./SponsorTokenExpires.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Savings/SponsorTokenExpires",
  component: SponsorTokenExpires,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    expiresSoon: { control: "boolean", description: "Whether sponsorship is expiring soon" },
    expiresAt: { control: "text", description: "Date when the sponsor token expires" },
    name: { control: "text", description: "Name of the sponsor token" },
  },
} as Meta<typeof SponsorTokenExpires>;

// Template for rendering the component
const Template: StoryFn<typeof SponsorTokenExpires> = (args) => ({
  components: { SponsorTokenExpires },
  setup() {
    return { args };
  },
  template: '<SponsorTokenExpires v-bind="args" />',
});

// Create stories for each variant
export const SomeDay = Template.bind({});
SomeDay.args = {
  status: {
    expiresSoon: true,
    expiresAt: new Date(Date.now() + 22 * 20 * 60 * 1000).toISOString(),
  },
};

export const Empty = Template.bind({});
Empty.args = {};
