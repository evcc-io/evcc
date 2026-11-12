import PriceChart from "./PriceChart.vue";
import { CURRENCY, type UiForecastSlot } from "@/types/evcc";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Forecast/PriceChart",
  component: PriceChart,
} as Meta<typeof PriceChart>;

const HOUR = 3600 * 1000;
const SLOT = HOUR / 4;
const HOURS = 96;
// estimates take over at 16:00, at least a day ahead
const KNOWN_HOURS = 24 + ((16 - new Date().getHours() + 24) % 24);

// the chart starts at the current hour
const start = () => new Date().setMinutes(0, 0, 0);

// daily price shape (0..1): low at night and noon, peaks in the morning and evening
const dayShape = (hour: number) =>
  0.35 +
  0.25 * Math.exp(-(((hour - 8) / 2) ** 2)) +
  0.65 * Math.exp(-(((hour - 19) / 1.5) ** 2)) -
  0.45 * Math.exp(-(((hour - 13) / 1.2) ** 2));

function slots(min: number, max: number, estimateAfterHours = Infinity): UiForecastSlot[] {
  const from = start();
  return Array.from({ length: HOURS * 4 }, (_, i) => {
    const t = from + i * SLOT;
    const d = new Date(t);
    // deterministic pseudo-random jitter per slot
    const n = Math.sin(i * 12.9898) * 43758.5453;
    const noise = ((n - Math.floor(n)) * 2 - 1) * 0.06;
    const estimate = i >= estimateAfterHours * 4;
    // estimates come from another source, so the curve jumps where they take over
    const shape = dayShape(d.getHours() + d.getMinutes() / 60) + (estimate ? 0.1 : 0);
    const value = min + (max - min) * (shape + noise);
    return {
      start: t,
      end: t + SLOT,
      value: Math.round(value * 1000) / 1000,
      ...(estimate ? { estimate } : {}),
    };
  });
}

interface Variant {
  grid: () => UiForecastSlot[];
  feedin?: () => UiForecastSlot[];
}

const import_ = () => slots(0.15, 0.35);
const export_ = () => slots(0.03, 0.08);
const importEstimate = () => slots(0.15, 0.35, KNOWN_HOURS);
const exportEstimate = () => slots(0.03, 0.08, KNOWN_HOURS);
// prices dip below zero around noon
const negativeEstimate = () => slots(-0.15, 0.3, KNOWN_HOURS);

const VARIANTS: Record<string, Variant> = {
  Import: { grid: import_ },
  ImportExport: { grid: import_, feedin: export_ },
  ImportEstimate: { grid: importEstimate },
  ImportEstimateExport: { grid: importEstimate, feedin: export_ },
  // estimates take over at different times
  BothEstimate: { grid: importEstimate, feedin: () => slots(0.03, 0.08, KNOWN_HOURS + 30) },
  ExportEstimate: { grid: import_, feedin: exportEstimate },
  AllEstimate: { grid: () => slots(0.15, 0.35, 0) },
  NegativeEstimate: { grid: negativeEstimate },
  NegativeEstimateExport: {
    grid: negativeEstimate,
    feedin: () => slots(-0.05, 0.1, KNOWN_HOURS),
  },
  AllNegativeEstimate: { grid: () => slots(-0.3, -0.05, KNOWN_HOURS) },
  // export crosses the import price
  OverlapEstimate: {
    grid: () => slots(0.1, 0.3, KNOWN_HOURS),
    feedin: () => slots(0.18, 0.22, KNOWN_HOURS),
  },
};

export const Overview: StoryFn<typeof PriceChart> = () => ({
  components: { PriceChart },
  setup() {
    return {
      charts: Object.entries(VARIANTS).map(([name, { grid, feedin }]) => ({
        name,
        grid: grid(),
        feedin: feedin?.(),
      })),
      currency: CURRENCY.EUR,
      endDate: new Date(start() + HOURS * HOUR),
    };
  },
  template: `<div class="p-4">
    <div v-for="c in charts" :key="c.name" class="mb-4">
      <h6 class="mb-0">{{ c.name }}</h6>
      <PriceChart :grid="c.grid" :feedin="c.feedin" :currency="currency" :chart-width="1800" :end-date="endDate" />
    </div>
  </div>`,
});
