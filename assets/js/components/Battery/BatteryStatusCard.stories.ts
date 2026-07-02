import BatteryStatusCard from "./BatteryStatusCard.vue";
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
