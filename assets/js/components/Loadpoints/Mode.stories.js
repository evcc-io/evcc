import Mode from "./Mode.vue";

export default {
  title: "Loadpoints/Mode",
  component: Mode,
  argTypes: {
    mode: {
      control: "select",
      options: ["off", "now", "minpv", "pv"],
      description: "Charging mode",
    },
    pvPossible: { control: "boolean", description: "Whether PV is possible" },
    hasSmartCost: { control: "boolean", description: "Whether smart cost is available" },
  },
  parameters: {
    layout: "centered",
  },
};

const createStory = (props) => {
  const story = (args) => ({
    components: { Mode },
    setup() {
      return { args };
    },
    template: '<Mode v-bind="args" />',
  });
  story.args = props;
  return story;
};

export const Minimal = createStory({ mode: "now" });

export const Full = createStory({
  mode: "pv",
  pvPossible: true,
  hasSmartCost: true,
});

export const SmartGridOnly = createStory({
  mode: "pv",
  pvPossible: false,
  hasSmartCost: true,
});
