import StatCards from "./StatCards.vue";
import Stat from "./Stat.vue";
import ForecastDeviation from "./ForecastDeviation.vue";
import formatter from "@/mixins/formatter";
import { groupColor } from "./groups";
import type { Meta, StoryFn } from "@storybook/vue3";

export default {
  title: "Energy/ForecastStat",
  component: StatCards,
} as Meta<typeof StatCards>;

interface Args {
  remaining?: number;
  tomorrow?: number;
  accuracy?: number;
  delta?: number;
  spans?: ([number, number] | null)[];
}

// 15 minute buckets of a sunny day, production off by `factor` around midday
const day = (factor: number): ([number, number] | null)[] =>
  Array.from({ length: 96 }, (_, i) => {
    const sun = Math.max(0, Math.sin(((i - 24) / 56) * Math.PI));
    const forecast = sun * 2;
    return [forecast, forecast * (1 + factor * Math.sin(i / 3))];
  });

const labels = Array.from({ length: 96 }, (_, i) => {
  const h = String(Math.floor(i / 4)).padStart(2, "0");
  const m = String((i % 4) * 15).padStart(2, "0");
  return `${h}:${m}`;
});

const Template: StoryFn<Args> = (args) => ({
  components: { StatCards, Stat, ForecastDeviation },
  mixins: [formatter],
  setup() {
    return { args, labels, color: groupColor("forecast") };
  },
  computed: {
    stats() {
      const kWh = (v: number) => this.fmtKWh(v);
      if (args.remaining !== undefined) {
        return [
          {
            key: "remaining",
            label: this.$t("energy.group.forecast"),
            number: args.remaining,
            format: kWh,
            sub: this.$t("energy.production.remainingToday"),
          },
        ];
      }
      const delta = args.delta ?? 0;
      return [
        {
          key: "forecast",
          label: this.$t("energy.production.accuracy"),
          number: args.accuracy,
          format: (v: number) => this.fmtPercentage(v),
          sub: this.$t(delta >= 0 ? "energy.production.above" : "energy.production.below", {
            value: this.fmtKWh(Math.abs(delta)),
          }),
        },
      ];
    },
  },
  template: `
    <!-- on large screens the cards take the height of the chart beside them -->
    <div class="vh-100 p-4">
      <div class="col-12 col-lg-3 h-50">
      <StatCards :stats="stats">
        <template v-if="args.tomorrow !== undefined" #remaining>
          <Stat
            class="mt-3"
            :number="args.tomorrow"
            :format="(v) => fmtKWh(v)"
            :sub="$t('energy.production.expectedTomorrow')"
          />
        </template>
        <template v-if="args.spans" #forecast>
          <ForecastDeviation class="mt-3" :spans="args.spans" :color="color" :labels="labels" power />
        </template>
      </StatCards>
      </div>
    </div>
  `,
});

export const Today = Template.bind({});
Today.args = { remaining: 22, tomorrow: 48 };

export const TodayWithoutTomorrow = Template.bind({});
TodayWithoutTomorrow.args = { remaining: 22 };

export const AccuracyAbove = Template.bind({});
AccuracyAbove.args = { accuracy: 87, delta: 3.2, spans: day(-0.15) };

export const AccuracyBelow = Template.bind({});
AccuracyBelow.args = { accuracy: 72, delta: -5.4, spans: day(0.3) };
