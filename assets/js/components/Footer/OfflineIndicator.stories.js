import OfflineIndicator from "./OfflineIndicator.vue";

export default {
  title: "Footer/OfflineIndicator",
  component: OfflineIndicator,
  argTypes: {
    offline: { control: "boolean" },
    fatal: { control: "object" },
  },
};

const Template = (args) => ({
  components: { OfflineIndicator },
  setup() {
    return { args };
  },
  template: '<OfflineIndicator v-bind="args" />',
});

export const Offline = Template.bind({});
Offline.args = {
  offline: true,
};

export const Fatal = Template.bind({});
Fatal.args = {
  fatal: {
    error: `cannot create charger 'wallbox_cphl': cannot create charger type 'template': cannot create
charger type 'hardybarth-ecbl: Post http://192.168.2.219/api/v1/chargecontrols/1/mode: context deadline exceeded (Client.Timeout
exceeded while awaiting headers)`,
  },
};
