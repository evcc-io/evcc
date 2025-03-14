import SponsorTokenExpires from "./SponsorTokenExpires.vue";

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
};

// Template for rendering the component
const Template = (args) => ({
  components: { SponsorTokenExpires },
  setup() {
    return { args };
  },
  template: '<SponsorTokenExpires v-bind="args" />',
});

// Create stories for each variant
export const SomeDay = Template.bind({});
SomeDay.args = {
  expiresSoon: true,
  expiresAt: new Date(Date.now() + 22 * 20 * 60 * 1000).toISOString(),
};

export const Empty = Template.bind({});
Empty.args = {};
