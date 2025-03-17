import LiveCommunity from "./LiveCommunity.vue";

export default {
  title: "Savings/LiveCommunity",
  component: LiveCommunity,
  parameters: {
    layout: "centered",
  },
};

const Template = () => ({
  components: { LiveCommunity },
  template: "<LiveCommunity />",
});

export const Default = Template.bind({});
