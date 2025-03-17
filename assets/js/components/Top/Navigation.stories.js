import Navigation from "./Navigation.vue";

export default {
  title: "Top/Navigation",
  component: Navigation,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    vehicleLogins: { control: "object" },
    sponsor: { control: "object" },
  },
};

const Template = (args) => ({
  components: { Navigation },
  setup() {
    return { args };
  },
  template: '<Navigation v-bind="args" />',
});

export const Standard = Template.bind({});
Standard.args = {};

export const VehicleLogins = Template.bind({});
VehicleLogins.args = {
  vehicleLogins: {
    "Mercedes EQS": {
      authenticated: true,
      uri: "https://login-provider-a.test/",
    },
    "Nissan Leaf Pro": {
      authenticated: true,
      uri: "https://login-provider-b.test/",
    },
  },
};

export const PendingVehicleLogins = Template.bind({});
PendingVehicleLogins.args = {
  vehicleLogins: {
    "Mercedes EQS": {
      authenticated: true,
      uri: "https://login-provider-a.test/",
    },
    "Nissan Leaf Pro": {
      authenticated: false,
      uri: "https://login-provider-b.test/",
    },
  },
};

export const TokenExpires = Template.bind({});
TokenExpires.args = {
  sponsor: { expiresSoon: true },
};
