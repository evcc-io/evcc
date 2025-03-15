import BatteryIcon from "./BatteryIcon.vue";

export default {
  title: "Energyflow/BatteryIcon",
  component: BatteryIcon,
  parameters: {
    layout: "centered",
  },
};

const createStory = (args) => {
  const story = () => ({
    components: { BatteryIcon },
    setup() {
      return { args };
    },
    template: '<BatteryIcon v-bind="args" />',
  });
  story.args = args;
  return story;
};

export const Empty = createStory({ soc: 0 });
export const Soc10 = createStory({ soc: 10 });
export const Soc20 = createStory({ soc: 20 });
export const Soc30 = createStory({ soc: 30 });
export const Soc40 = createStory({ soc: 40 });
export const Soc50 = createStory({ soc: 50 });
export const Soc60 = createStory({ soc: 60 });
export const Soc70 = createStory({ soc: 70 });
export const Soc80 = createStory({ soc: 80 });
export const Soc90 = createStory({ soc: 90 });
export const Hold = createStory({ hold: true });
export const GridCharge = createStory({ gridCharge: true });
