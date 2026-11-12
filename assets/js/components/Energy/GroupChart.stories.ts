import GroupChart from "./GroupChart.vue";
import { groupColor } from "./groups";
import { PERIODS } from "../Sessions/types";
import { CURRENCY } from "@/types/evcc";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Energy/GroupChart",
  component: GroupChart,
} as Meta<typeof GroupChart>;

interface Args {
  currency?: CURRENCY;
  minPrice?: number;
  maxPrice?: number;
}

const from = new Date(2026, 0, 1);
const to = new Date(2027, 0, 1);
const months = Array.from({ length: 12 }, (_, i) => i);

// a year of grid import, high in winter
const series = [
  {
    title: "Grid",
    group: "grid",
    data: months.map((i) => ({
      start: new Date(2026, i, 1).toISOString(),
      end: new Date(2026, i + 1, 1).toISOString(),
      energy: 250 + 200 * Math.cos((i / 12) * 2 * Math.PI),
      returnEnergy: 0,
    })),
  },
];

const Template: StoryFn<Args> = (args) => ({
  components: { GroupChart },
  setup() {
    return { args, series, from, to, period: PERIODS.YEAR, color: groupColor("grid") };
  },
  computed: {
    prices() {
      const { minPrice, maxPrice } = args;
      if (minPrice === undefined || maxPrice === undefined) return null;
      const avg = months.map((i) => minPrice + ((maxPrice - minPrice) * i) / 11);
      return { import: { avg, lo: avg, hi: avg } };
    },
  },
  // the border marks the chart's own box, long panel labels grow out of it
  template: `
    <div class="p-4">
      <GroupChart class="border" group="grid" singleValue :color="color" :series="series" :period="period"
        :from="from" :to="to" :prices="prices" :currency="args.currency" />
    </div>
  `,
});

// the reference for the plot's edges, they must not move with a panel below the bars
export const Default = Template.bind({});
Default.args = {};

export const PricePanel = Template.bind({});
PricePanel.args = { currency: CURRENCY.EUR, minPrice: 0.284, maxPrice: 0.521 };

// a currency without subunit shows three decimals, the labels grow out of the chart
export const PricePanelLongLabels = Template.bind({});
PricePanelLongLabels.args = { currency: CURRENCY.CZK, minPrice: 14.123, maxPrice: 27.345 };
