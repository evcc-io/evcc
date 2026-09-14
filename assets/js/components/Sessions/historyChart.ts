import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import echartsChart from "@/mixins/echartsChart";
import { PERIODS, type Session } from "./types";

// chartOption is provided by each consuming component's computed
type WithChartOption = { chartOption: Record<string, unknown> };

// shared date bucketing and axis formatting for the history bar charts
export default defineComponent({
  mixins: [formatter, echartsChart],
  props: {
    sessions: { type: Array as PropType<Session[]>, default: () => [] },
    period: { type: String as PropType<PERIODS>, default: PERIODS.TOTAL },
  },
  computed: {
    firstDay(): Date | null {
      if (this.sessions.length === 0) {
        return null;
      }
      return new Date(this.sessions[0]!.created);
    },
    lastDay(): Date | null {
      if (this.sessions.length === 0) {
        return null;
      }
      return new Date(this.sessions[this.sessions.length - 1]!.created);
    },
    month(): number {
      return (this.firstDay?.getMonth() || 0) + 1;
    },
    year(): number {
      return this.firstDay?.getFullYear() || 0;
    },
    // bucket index range [from, to] for the selected period
    bucketRange(): [number, number] | null {
      if (!this.firstDay || !this.lastDay) return null;
      if (this.period === PERIODS.TOTAL) {
        return [this.firstDay.getFullYear(), this.lastDay.getFullYear()];
      }
      if (this.period === PERIODS.YEAR) {
        return [1, 12];
      }
      return [1, new Date(this.lastDay.getFullYear(), this.lastDay.getMonth() + 1, 0).getDate()];
    },
    tooltipHead(): (label: string) => string {
      if (this.period === PERIODS.TOTAL) {
        return (label) => label;
      }
      if (this.period === PERIODS.YEAR) {
        return (label) => this.fmtMonth(new Date(this.year, Number(label) - 1, 1), false);
      }
      return (label) => this.fmtDayMonth(new Date(this.year, this.month - 1, Number(label)));
    },
  },
  methods: {
    bucketIndex(date: Date): number {
      if (this.period === PERIODS.MONTH) return date.getDate();
      if (this.period === PERIODS.YEAR) return date.getMonth() + 1;
      return date.getFullYear();
    },
    xAxisLabel(label: string): string {
      return this.period === PERIODS.YEAR
        ? this.fmtMonth(new Date(this.year, Number(label) - 1, 1), true)
        : label;
    },
    applyChartOption() {
      this.chart?.setOption((this as unknown as WithChartOption).chartOption, {
        notMerge: true,
      });
    },
  },
});
