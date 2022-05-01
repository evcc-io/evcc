import "bootstrap";
import TopNavigation from "./TopNavigation.vue";

export default {
  title: "Main/TopNavigation",
  component: TopNavigation,
  parameters: {
    backgrounds: {
      default: "light",
    },
  },
};

const Template = (args) => ({
  setup() {
    return { args };
  },
  components: { TopNavigation },
  template: `<div class="d-flex justify-content-end" style="padding-bottom: 18rem">
    <TopNavigation v-bind="args"></TopNavigation>
  </div>`,
});

export const Default = Template.bind({});
Default.args = {};

export const VehicleLogins = Template.bind({});
VehicleLogins.args = {
  vehicleLogins: {
    ["Mercedes EQS"]: {
      authenticated: true,
      uri: "https://login-provider-a.test/",
    },
    ["Nissan Leaf Pro"]: {
      authenticated: false,
      uri: "https://login-provider-b.test/",
    },
  },
};

export const PendingVehicleLogins = Template.bind({});
PendingVehicleLogins.args = {
  vehicleLogins: {
    ["Mercedes EQS"]: {
      authenticated: true,
      uri: "https://login-provider-a.test/",
    },
    ["Nissan Leaf Pro"]: {
      authenticated: true,
      uri: "https://login-provider-b.test/",
    },
  },
};
