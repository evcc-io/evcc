import Mode from "./Mode.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Loadpoints/Mode",
  component: Mode,
  argTypes: {
    mode: {
      control: "select",
      options: ["off", "smart", "now"],
      description: "Charging mode",
    },
    pvPossible: { control: "boolean", description: "Whether PV is possible" },
    smartCostAvailable: { control: "boolean", description: "Whether smart cost is available" },
    alwaysCharge: {
      control: "select",
      options: ["off", "on", "once"],
      description: "Always charge state",
    },
    switchDevice: { control: "boolean", description: "Charger without current control" },
    continuous: { control: "boolean", description: "Continuously operating heatpump" },
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
  mode: "smart",
  pvPossible: true,
  smartCostAvailable: true,
  effectiveMinCurrent: 6,
};

export const SmartGridOnly = Template.bind({});
SmartGridOnly.args = {
  mode: "smart",
  pvPossible: false,
  smartCostAvailable: true,
  effectiveMinCurrent: 6,
};

export const AlwaysCharge = Template.bind({});
AlwaysCharge.args = {
  mode: "smart",
  pvPossible: true,
  alwaysCharge: "on",
  effectiveMinCurrent: 6,
};

export const AlwaysChargeOnce = Template.bind({});
AlwaysChargeOnce.args = {
  mode: "smart",
  pvPossible: true,
  alwaysCharge: "once",
  effectiveMinCurrent: 6,
};

export const SwitchDevice = Template.bind({});
SwitchDevice.args = {
  mode: "smart",
  pvPossible: true,
  switchDevice: true,
};

export const ContinuousHeatpump = Template.bind({});
ContinuousHeatpump.args = {
  mode: "smart",
  pvPossible: true,
  continuous: true,
  heating: true,
  alwaysCharge: "on",
  effectiveMinCurrent: 6,
};
