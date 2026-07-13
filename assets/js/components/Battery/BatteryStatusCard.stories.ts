import BatteryStatusCard from "./BatteryStatusCard.vue";
import { BATTERY_MODE } from "@/types/evcc";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Battery/BatteryStatusCard",
  component: BatteryStatusCard,
  parameters: {
    layout: "centered",
  },
  argTypes: {
    title: { control: "text" },
    soc: { control: { type: "range", min: 0, max: 100 } },
    power: { control: "number", description: "W, + discharging / - charging" },
    capacity: { control: "number", description: "kWh, 0 = unspecified" },
    color: { control: "color" },
  },
} as Meta<typeof BatteryStatusCard>;

// narrow: how the card looks as a column in a multi-battery grid
const Narrow: StoryFn<typeof BatteryStatusCard> = (args) => ({
  components: { BatteryStatusCard },
  setup() {
    return { args };
  },
  template: '<div style="width: 420px"><BatteryStatusCard v-bind="args" /></div>',
});

const base = { title: "Sungrow", capacity: 13.5, color: "#0BA631" };

export const Charging = Narrow.bind({});
Charging.args = { ...base, soc: 76, power: -800 };

export const Discharging = Narrow.bind({});
Discharging.args = {
  ...base,
  title: "Anker",
  capacity: 7.5,
  color: "#7FC41B",
  soc: 40,
  power: 1200,
};

export const Idle = Narrow.bind({});
Idle.args = { ...base, title: "Battery", capacity: 13.4, soc: 98, power: 0 };

export const Empty = Narrow.bind({});
Empty.args = { ...base, title: "Battery", capacity: 13.4, soc: 4, power: -600 };

export const NoCapacity = Narrow.bind({});
NoCapacity.args = { ...base, title: "Battery", capacity: 0, soc: 55, power: 300 };

// optimizer states, stacked below the values (as on the combined card)
export const OptimizerNormal = Narrow.bind({});
OptimizerNormal.args = { ...base, soc: 62, power: 900, suggestion: { action: "normal" } };

export const OptimizerHold = Narrow.bind({});
OptimizerHold.args = { ...base, soc: 88, power: 0, suggestion: { action: "hold" } };

export const OptimizerCharge = Narrow.bind({});
OptimizerCharge.args = {
  ...base,
  soc: 48,
  power: 2500,
  suggestion: { action: "charge" },
};

export const OptimizerHoldCharge = Narrow.bind({});
OptimizerHoldCharge.args = { ...base, soc: 70, power: 0, suggestion: { action: "holdcharge" } };

export const WithForecast = Narrow.bind({});
WithForecast.args = {
  ...base,
  soc: 62,
  power: 900,
  forecast: {
    highest: { soc: 100, time: "2026-07-01T16:30:00+02:00" },
    lowest: { soc: 12, time: "2026-07-02T06:00:00+02:00" },
  },
};

export const SuggestionAndForecast = Narrow.bind({});
SuggestionAndForecast.args = {
  ...base,
  soc: 48,
  power: 2500,
  suggestion: { action: "charge" },
  forecast: { highest: { soc: 100, time: "2026-07-01T18:00:00+02:00", limit: true } },
};

// grid of cards whose current mode (gauge) deliberately differs from the suggested action,
// so the suggestion icon always signals a change.
// current mode: charge/holdcharge/hold are locked modes; discharging is power-derived (no mode)
const cards = [
  {
    title: "Sungrow",
    soc: 76,
    power: 0,
    capacity: 13.5,
    color: "#0BA631",
    controllable: true,
    batteryMode: BATTERY_MODE.CHARGE, // current: charge
    suggestion: { action: "normal", actionable: true },
  },
  {
    title: "Anker",
    soc: 40,
    power: 1200, // current: discharging
    capacity: 7.5,
    color: "#7FC41B",
    controllable: true,
    suggestion: { action: "hold", actionable: true },
  },
  {
    title: "Fox ESS",
    soc: 88,
    power: 0,
    capacity: 10.4,
    color: "#0FD0BF",
    controllable: true,
    batteryMode: BATTERY_MODE.HOLDCHARGE, // current: holdcharge
    suggestion: { action: "hold", actionable: true },
  },
  {
    title: "Huawei",
    soc: 30,
    power: 0,
    capacity: 5,
    color: "#4EABE6",
    controllable: true,
    batteryMode: BATTERY_MODE.HOLD, // current: hold
    suggestion: { action: "charge", actionable: true },
  },
];

export const CurrentVsSuggested = () => ({
  components: { BatteryStatusCard },
  setup() {
    return { cards };
  },
  template: `<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(380px, 1fr)); gap: 1rem">
    <BatteryStatusCard v-for="c in cards" :key="c.title" v-bind="c" />
  </div>`,
});
