import TimeSeriesDataTable from "./TimeSeriesDataTable.vue";
import Card from "../Helper/Card.vue";
import { CURRENCY } from "@/types/evcc";
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
  title: "Optimize/TimeSeriesDataTable",
  component: TimeSeriesDataTable,
  parameters: {
    layout: "padded",
  },
} as Meta<typeof TimeSeriesDataTable>;

const base = {
  mode: "request" as const,
  batteryDetails,
  batteryColors,
  demandColors,
  timestamps,
  currency: CURRENCY.EUR,
};

const Template: StoryFn<typeof TimeSeriesDataTable> = (args) => ({
  components: { TimeSeriesDataTable, Card },
  setup() {
    return { args };
  },
  template: `<div class="container px-0"><Card title="Environment" subtitle="64 steps · 16 h horizon" edge-to-edge class="box-pull-out"><TimeSeriesDataTable v-bind="args" /></Card></div>`,
});

// nothing to break down, the backend sends no details
export const BaseLoadOnly = Template.bind({});
BaseLoadOnly.args = { ...base, evopt: evopt([home]), demandDetails: [] };

// heating loadpoints as indented rows under the total
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
