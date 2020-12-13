import Toasts from "./Toasts.vue";

export default {
  title: "Main/Toasts",
  component: Toasts,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { Toasts },
  template: '<Toasts v-bind="$props"></Toasts>',
});

export const Base = Template.bind({});
Base.args = {
  items: {
    id: 1,
    message: "Evil warning",
    type: "warn",
  },
};
