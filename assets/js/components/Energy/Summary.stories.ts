import Summary from "./Summary.vue";
import type { Meta, StoryFn } from "@storybook/vue3";
import type { Flow, FlowCo2, FlowCost } from "./types";

export default {
  title: "Energy/Summary",
  component: Summary,
  argTypes: {
    autarky: { control: { type: "range", min: 0, max: 100 } },
    production: { control: "number", description: "kWh, metered" },
    consumption: { control: "number", description: "kWh, metered" },
    gridImport: { control: "number", description: "kWh, metered" },
    hasBattery: { control: "boolean" },
    hasPriceTariff: { control: "boolean" },
    hasCo2Tariff: { control: "boolean" },
  },
} as Meta<typeof Summary>;

const Template: StoryFn<typeof Summary> = (args) => ({
  components: { Summary },
  setup() {
    return { args };
  },
  template: '<Summary v-bind="args" />',
});

// own energy at the feed-in price, grid energy at the grid price
const FEED_IN = 0.11;
const GRID = 0.29;
const flow = (from: Flow["from"], to: Flow["to"], energy: number): Flow => {
  const priced = to === "home" || to === "loadpoint";
  const price = from === "grid" ? GRID : FEED_IN;
  return { from, to, energy, ...(priced ? { cost: energy * price, pricedEnergy: energy } : {}) };
};

// a year of a site with solar, battery and two chargers
const flows: Flow[] = [
  flow("pv", "home", 3600),
  flow("pv", "loadpoint", 1500),
  flow("pv", "battery", 1700),
  flow("pv", "export", 2700),
  flow("battery", "home", 1500),
  flow("grid", "home", 500),
  flow("grid", "loadpoint", 200),
];
const cost: FlowCost = {
  import: 700 * GRID,
  export: 2700 * FEED_IN,
  importEnergy: 700,
  exportEnergy: 2700,
  avgGrid: GRID,
};
const co2: FlowCo2 = { consumption: 231, consumptionEnergy: 7300, baseline: 2409, avgCo2: 330 };

const base = {
  autarky: 90,
  production: 9500,
  consumption: 7300,
  gridImport: 700,
  flows,
  cost,
  co2,
  hasBattery: true,
  hasPriceTariff: true,
  hasCo2Tariff: true,
};

export const Year = Template.bind({});
Year.args = base;

// the battery rows are left out, grid export and grid stay
const withoutBattery = flows.filter((f) => f.from !== "battery" && f.to !== "battery");
export const NoBattery = Template.bind({});
NoBattery.args = {
  ...base,
  hasBattery: false,
  flows: withoutBattery,
  production: 7800,
  consumption: 5800,
  autarky: 88,
};

// a battery that did nothing still has its rows, at zero
export const IdleBattery = Template.bind({});
IdleBattery.args = { ...NoBattery.args, hasBattery: true };

// nothing came from the grid: the row stays at zero
export const NoGridImport = Template.bind({});
NoGridImport.args = {
  ...base,
  flows: flows.filter((f) => f.from !== "grid"),
  consumption: 6600,
  gridImport: 0,
  autarky: 100,
  cost: { ...cost, import: 0, importEnergy: 0 },
};

// no tariff set up: the tiles point to the settings
export const NoTariffs = Template.bind({});
NoTariffs.args = {
  ...base,
  cost: undefined,
  co2: undefined,
  hasPriceTariff: false,
  hasCo2Tariff: false,
};

// tariffs are set up, this period has no data for them: no link
export const TariffsWithoutData = Template.bind({});
TariffsWithoutData.args = { ...base, cost: undefined, co2: undefined };

// a year of a large site: farm roof, several chargers. The widest values the tiles
// have to hold
export const LargeNumbers = Template.bind({});
LargeNumbers.args = {
  ...base,
  autarky: 68,
  production: 186400,
  consumption: 143750,
  gridImport: 45980,
  flows: [
    flow("pv", "home", 61200),
    flow("pv", "loadpoint", 24350),
    flow("pv", "battery", 14900),
    flow("pv", "export", 85950),
    flow("battery", "home", 12220),
    flow("grid", "home", 31480),
    flow("grid", "loadpoint", 14500),
  ],
  cost: {
    import: 45980 * GRID,
    export: 85950 * FEED_IN,
    importEnergy: 45980,
    exportEnergy: 85950,
    avgGrid: GRID,
  },
  co2: { consumption: 15173, consumptionEnergy: 143750, baseline: 47438, avgCo2: 330 },
};

// a single day, small numbers
export const Day = Template.bind({});
Day.args = {
  ...base,
  autarky: 74,
  production: 53.4,
  consumption: 41.7,
  gridImport: 10.8,
  flows: [
    flow("pv", "home", 21.3),
    flow("pv", "loadpoint", 8.1),
    flow("pv", "battery", 20.8),
    flow("pv", "export", 3.2),
    flow("battery", "home", 1.5),
    flow("grid", "home", 2.4),
    flow("grid", "loadpoint", 8.4),
  ],
  cost: { import: 3.13, export: 0.35, importEnergy: 10.8, exportEnergy: 3.2, avgGrid: GRID },
  co2: { consumption: 3.6, consumptionEnergy: 41.7, baseline: 13.8, avgCo2: 330 },
};
