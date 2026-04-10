import { defineComponent, markRaw } from "vue";
import { echarts } from "./echarts";
import "./chartStyles.css";

// chartOption is provided by each consuming component's computed
type WithChartOption = { chartOption: Record<string, unknown> };

export default defineComponent({
  props: {
    chartWidth: { type: Number, required: true },
    endDate: { type: Date, required: true },
    scrollLeft: { type: Number, default: 0 },
  },
  emits: ["scroll"],
  data(): {
    chart: echarts.ECharts | null;
    startDate: Date;
    tooltipVisible: boolean;
  } {
    return {
      chart: null,
      tooltipVisible: false,
      startDate: new Date(),
    };
  },
  watch: {
    chartOption: {
      handler() {
        this.chart?.setOption((this as unknown as WithChartOption).chartOption);
      },
      deep: true,
    },
    scrollLeft(val: number) {
      const el = this.$refs["scrollEl"] as HTMLElement;
      if (el && Math.abs(el.scrollLeft - val) > 1) {
        el.scrollLeft = val;
      }
    },
  },
  mounted() {
    this.updateStartDate();
    this.initChart();
  },
  beforeUnmount() {
    this.chart?.dispose();
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
    initChart() {
      const el = this.$refs["chartEl"] as HTMLElement;
      this.chart = markRaw(echarts.init(el));
      this.chart.setOption((this as unknown as WithChartOption).chartOption);
      this.chart.on("showTip", () => {
        this.tooltipVisible = true;
      });
      this.chart.on("hideTip", () => {
        this.tooltipVisible = false;
      });
      const resetTouch = () => {
        this.chart?.dispatchAction({ type: "hideTip" });
        this.tooltipVisible = false;
      };
      el.addEventListener("touchend", resetTouch);
      el.addEventListener("touchcancel", resetTouch);
    },
  },
});
