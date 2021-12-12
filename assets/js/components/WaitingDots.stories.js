import WaitingDots from "./WaitingDots.vue";

export default {
  title: "Main/WaitingDots",
  component: WaitingDots,
  argTypes: {},
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { WaitingDots },
  template: '<WaitingDots v-bind="$props"></WaitingDots>',
});

export const HorizontalUp = Template.bind({});
HorizontalUp.args = {
  orientation: "horizontal",
  direction: "up",
};
export const HorizontalDown = Template.bind({});
HorizontalDown.args = {
  orientation: "horizontal",
  direction: "down",
};
export const VerticalUp = Template.bind({});
VerticalUp.args = {
  orientation: "vertical",
  direction: "up",
};
export const VerticalDown = Template.bind({});
VerticalDown.args = {
  orientation: "vertical",
  direction: "down",
};
