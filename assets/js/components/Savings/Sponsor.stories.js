import Sponsor from "./Sponsor.vue";

export default {
  title: "Savings/Sponsor",
  component: Sponsor,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    name: {
      control: "text",
      description: "Sponsor name (use 'trial' or 'victron' for special states)",
    },
    expiresAt: {
      control: "text",
      description: "When the sponsorship expires (ISO date string)",
    },
    expiresSoon: {
      control: "boolean",
      description: "Whether the sponsorship is expiring soon",
    },
  },
};

// Template for rendering the component
const Template = (args) => ({
  components: { Sponsor },
  setup() {
    return { args };
  },
  template: '<Sponsor v-bind="args" />',
});

// Create stories for each variant
export const NoSponsor = Template.bind({});
NoSponsor.args = {};

export const Trial = Template.bind({});
Trial.args = {
  name: "trial",
};

export const IndividualSponsor = Template.bind({});
IndividualSponsor.args = {
  name: "naltatis",
};

export const VictronDevice = Template.bind({});
VictronDevice.args = {
  name: "victron",
};

// Add an extra story showing the expiring state
export const ExpiringSponsor = Template.bind({});
ExpiringSponsor.args = {
  name: "naltatis",
  expiresSoon: true,
  expiresAt: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(), // 7 days from now
};
