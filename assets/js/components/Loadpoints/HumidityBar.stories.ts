import HumidityBar from "./HumidityBar.vue";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Loadpoints/HumidityBar",
  component: HumidityBar,
  argTypes: {
    humidity: { control: "number" },
    targetHumidity: { control: "number" },
    enabled: { control: "boolean" },
  },
} as Meta<typeof HumidityBar>;

const Template: StoryFn<typeof HumidityBar> = (args) => ({
  components: { HumidityBar },
  setup() {
    return { args };
  },
  template:
    '<div style="width: 340px; padding: 1rem; background: var(--evcc-box);"><HumidityBar v-bind="args" /></div>',
});

export const Active = Template.bind({});
Active.args = { humidity: 63, targetHumidity: 55, enabled: true };

export const SensorUnavailable = Template.bind({});
SensorUnavailable.args = { humidity: null, targetHumidity: 55, enabled: false };
