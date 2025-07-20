import Mode from "./Mode.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

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
    smartCostAvailable: { control: "boolean", description: "Whether smart cost is available" },
  },
  parameters: {
    layout: "centered",
  },
} as Meta<typeof Mode>;

const Template: StoryFn<typeof Mode> = (args) => {
  const story = () => ({
    components: { Mode },
    setup() {
      return { args };
    },
    template: '<Mode v-bind="args" />',
  });
  story.args = args;
  return story;
};

export const Minimal = Template.bind({});
Minimal.args = { mode: "now" };

export const Full = Template.bind({});
Full.args = {
  mode: "pv",
  pvPossible: true,
  smartCostAvailable: true,
};

export const SmartGridOnly = Template.bind({});
SmartGridOnly.args = {
  mode: "pv",
  pvPossible: false,
  smartCostAvailable: true,
};
