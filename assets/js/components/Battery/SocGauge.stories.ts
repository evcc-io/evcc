import SocGauge from "./SocGauge.vue";
import colors, { batteryColor } from "@/colors";
import { BATTERY_MODE } from "@/types/evcc";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Battery/SocGauge",
  component: SocGauge,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    soc: { control: { type: "range", min: 0, max: 100 } },
    color: { control: "inline-radio", options: colors.batteryPalette },
    power: { control: "number", description: "W, + discharging / - charging" },
    mode: { control: "inline-radio", options: Object.values(BATTERY_MODE) },
  },
} as Meta<typeof SocGauge>;

const Template: StoryFn<typeof SocGauge> = (args) => ({
  components: { SocGauge },
  setup() {
    return { args };
  },
  template: '<SocGauge v-bind="args" />',
});

export const Default = Template.bind({});
Default.args = { soc: 60, color: batteryColor(0), power: -800 };

// (power, mode) pairs that each drive a distinct SocGauge status
const scenarios = [
  { label: "idle", power: 0, mode: undefined },
  { label: "charging", power: -800, mode: undefined },
  { label: "discharging", power: 800, mode: undefined },
  { label: "hold", power: 0, mode: BATTERY_MODE.HOLD },
  { label: "holdcharge", power: 0, mode: BATTERY_MODE.HOLDCHARGE },
  { label: "charge", power: -800, mode: BATTERY_MODE.CHARGE },
];

export const AllStates = () => ({
  components: { SocGauge },
  setup() {
    return { scenarios, color: batteryColor(0) };
  },
  template: `
    <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 24px;">
      <div v-for="s in scenarios" :key="s.label" style="display: flex; flex-direction: column; align-items: center; gap: 8px;">
        <SocGauge :soc="60" :color="color" :power="s.power" :mode="s.mode" />
        <small style="font-family: monospace; color: #666; font-size: 12px;">{{ s.label }}</small>
      </div>
    </div>
  `,
});
