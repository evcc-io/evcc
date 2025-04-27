import Phases from "./Phases.vue";

export default {
  title: "Loadpoints/Phases",
  component: Phases,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    phasesActive: { control: { type: "number" } },
    minCurrent: { control: { type: "number" } },
    maxCurrent: { control: { type: "number" } },
    offeredCurrent: { control: { type: "number" } },
    chargeCurrents: { control: { type: "object" } },
  },
};

const Template = (args) => ({
  components: { Phases },
  setup() {
    return { args };
  },
  template: '<Phases v-bind="args" />',
});

export const OnePhase = Template.bind({});
OnePhase.args = {
  phasesActive: 1,
  minCurrent: 6,
  maxCurrent: 16,
  offeredCurrent: 8,
  chargeCurrents: null,
};

export const TwoPhase = Template.bind({});
TwoPhase.args = {
  ...OnePhase.args,
  phasesActive: 2,
};

export const ThreePhase = Template.bind({});
ThreePhase.args = {
  ...OnePhase.args,
  phasesActive: 3,
};

export const RealCurrents = Template.bind({});
RealCurrents.args = {
  ...OnePhase.args,
  phasesActive: 3,
  offeredCurrent: 13,
  chargeCurrents: [11, 9, 12],
};

export const OnePhaseMoreAvailable = Template.bind({});
OnePhaseMoreAvailable.args = {
  ...OnePhase.args,
  offeredCurrent: 12,
  chargeCurrents: [6, 0.2, 0],
};

export const TwoPhasesActive = Template.bind({});
TwoPhasesActive.args = {
  ...OnePhase.args,
  phasesActive: 2,
  offeredCurrent: 16,
  chargeCurrents: [16, 16, 0.3],
};

export const AsymetricPhases = Template.bind({});
AsymetricPhases.args = {
  ...OnePhase.args,
  phasesActive: 2,
  offeredCurrent: 16,
  chargeCurrents: [8, 0.9, 14],
};

export const OnlySecondPhase = Template.bind({});
OnlySecondPhase.args = {
  ...OnePhase.args,
  phasesActive: 1,
  offeredCurrent: 13,
  chargeCurrents: [0, 13, 0],
};

export const MainlyThirdPhase = Template.bind({});
MainlyThirdPhase.args = {
  ...OnePhase.args,
  phasesActive: 1,
  offeredCurrent: 10,
  chargeCurrents: [0.007, 0.009, 5.945],
  minCurrent: 6,
  maxCurrent: 20,
};
