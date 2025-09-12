import Navigation from "./Navigation.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Top/Navigation",
  component: Navigation,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    authProviders: { control: "object" },
    sponsor: { control: "object" },
  },
} as Meta<typeof Navigation>;

const Template: StoryFn<typeof Navigation> = (args) => ({
  components: { Navigation },
  setup() {
    return { args };
  },
  template: '<Navigation v-bind="args" />',
});

export const Standard = Template.bind({});
Standard.args = {};

export const OAuthStatus = Template.bind({});
OAuthStatus.args = {
  authProviders: {
    "Mercedes EQS": {
      authenticated: true,
      id: "mercedes-eqs-9oqwjdf9oqwjd",
    },
    "Nissan Leaf Pro": {
      authenticated: true,
      id: "nissan-leaf-pro-9oqwjdf9oqwjd",
    },
  },
};

export const PendingOAuthStatus = Template.bind({});
PendingOAuthStatus.args = {
  authProviders: {
    "Mercedes EQS": {
      authenticated: true,
      id: "mercedes-eqs-9oqwjdf9oqwjd",
    },
    "Nissan Leaf Pro": {
      authenticated: false,
      id: "nissan-leaf-pro-9oqwjdf9oqwjd",
    },
  },
};

export const TokenExpires = Template.bind({});
TokenExpires.args = {
  sponsor: {
    name: "Sponsor",
    expiresAt: new Date().toISOString(),
    expiresSoon: true,
    fromYaml: false,
  },
};
