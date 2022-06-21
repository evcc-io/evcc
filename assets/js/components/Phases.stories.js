import Phases from "./Phases.vue";

export default {
  title: "Main/Phases",
  component: Phases,
  argTypes: {},
  parameters: { backgrounds: { default: "box" } },
};

const Template = (args, { argTypes }) => ({
  props: Object.keys(argTypes),
  components: { Phases },
  template: '<Phases v-bind="$props"></Phases>',
});

export const Base = Template.bind({});
Base.args = {
  activePhases: 1,
  minCurrent: 6,
  maxCurrent: 32,
  chargeCurrent: 10,
};

export const OnePhase = Template.bind({});
OnePhase.args = {
  activePhases: 1,
  minCurrent: 6,
  maxCurrent: 16,
  chargeCurrent: 6,
};

export const TwoPhase = Template.bind({});
TwoPhase.args = {
  activePhases: 2,
  minCurrent: 6,
  maxCurrent: 16,
  chargeCurrent: 8,
};

export const ThreePhases = Template.bind({});
ThreePhases.args = {
  activePhases: 3,
  minCurrent: 6,
  maxCurrent: 16,
  chargeCurrent: 12,
};

export const RealCurrents = Template.bind({});
RealCurrents.args = {
  activePhases: 3,
  minCurrent: 6,
  maxCurrent: 32,
  chargeCurrent: 16.5,
  chargeCurrents: [15.2, 15.76, 14.5],
};
