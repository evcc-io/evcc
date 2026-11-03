import { defineComponent } from "vue";
import echartsChart from "@/mixins/echartsChart";
import "./chartStyles.css";

export default defineComponent({
  mixins: [echartsChart],
  props: {
    chartWidth: { type: Number, required: true },
    endDate: { type: Date, required: true },
    scrollLeft: { type: Number, default: 0 },
  },
  emits: ["scroll"],
  data(): {
    startDate: Date;
    tooltipVisible: boolean;
  } {
    return {
      tooltipVisible: false,
      startDate: new Date(),
    };
  },
  watch: {
    scrollLeft(val: number) {
      const el = this.$refs["scrollEl"] as HTMLElement;
      if (el && Math.abs(el.scrollLeft - val) > 1) {
        el.scrollLeft = val;
      }
    },
  },
  created() {
    // before the first chart render so chartOption sees the rounded date
    this.updateStartDate();
  },
  methods: {
    updateStartDate() {
      const now = new Date();
      now.setMinutes(0, 0, 0);
      this.startDate = now;
    },
    onScroll(e: Event) {
      this.$emit("scroll", (e.target as HTMLElement).scrollLeft);
    },
    onChartInit() {
      this.chart?.on("showTip", () => {
        this.tooltipVisible = true;
      });
      this.chart?.on("hideTip", () => {
        this.tooltipVisible = false;
      });
    },
    onTouchTooltipReset() {
      this.tooltipVisible = false;
    },
  },
});
