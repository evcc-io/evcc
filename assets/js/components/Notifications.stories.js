import Notifications from "./Notifications.vue";
import i18n from "../i18n";

export default {
  title: "Main/Notifications",
  component: Notifications,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  i18n,
  props: Object.keys(argTypes),
  components: { Notifications },
  template: '<Notifications v-bind="$props"></Notifications>',
});

export const Base = Template.bind({});
Base.args = {
  notifications: [
    {
      message: "Server unavailable",
      type: "error",
    },
    {
      message: "charger out of sync: expected disabled, got enabled",
      type: "warn",
    },
    {
      message:
        "Amet irure quis incididunt voluptate esse. Commodo ea sunt est ipsum tempor nisi laboris voluptate labore elit laborum. Ex irure commodo reprehenderit consequat consequat do ad tempor aliquip deserunt eu. Laboris minim nostrud quis nisi. Dolor occaecat reprehenderit velit dolore exercitation cupidatat et voluptate. Nulla pariatur deserunt esse minim nisi nisi nulla. Sit eiusmod do incididunt sint minim pariatur aute.",
      type: "warn",
    },
  ],
};
