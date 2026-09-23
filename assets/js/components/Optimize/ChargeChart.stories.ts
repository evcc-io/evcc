import ChargeChart from "./ChargeChart.vue";
import Card from "../Helper/Card.vue";
import type { Meta, StoryFn } from "@storybook/vue3";
import {
  batteryColors,
  batteryDetails,
  carport,
  demandColors,
  evopt,
  heatPump,
  heatingRod,
  home,
  timestamps,
} from "./stories.fixture";

export default {
  title: "Optimize/ChargeChart",
  component: ChargeChart,
  parameters: {
    layout: "padded",
  },
} as Meta<typeof ChargeChart>;

const base = {
  batteryDetails,
  batteryColors,
  demandColors,
  timestamp: timestamps[0],
};

const Template: StoryFn<typeof ChargeChart> = (args) => ({
  components: { ChargeChart, Card },
  setup() {
    return { args };
  },
  template: `<div class="container px-0"><Card title="Charging Plan" subtitle="kW" edge-to-edge class="box-pull-out"><ChargeChart v-bind="args" /></Card></div>`,
});

// nothing to break down, the backend sends no details
export const BaseLoadOnly = Template.bind({});
BaseLoadOnly.args = { ...base, evopt: evopt([home]), demandDetails: [] };

// heating loadpoints stacked on the base load
export const HeatingLoadpoints = Template.bind({});
HeatingLoadpoints.args = {
  ...base,
  evopt: evopt([home, heatPump, heatingRod]),
  demandDetails: [home, heatPump, heatingRod],
};

// loadpoint without capacity decays as unmodelled load
export const HeatingAndUnmodelled = Template.bind({});
HeatingAndUnmodelled.args = {
  ...base,
  evopt: evopt([home, carport, heatPump]),
  demandDetails: [home, carport, heatPump],
};
