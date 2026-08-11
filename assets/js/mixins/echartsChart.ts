import { defineComponent, markRaw } from "vue";
import { echarts, registerTouchTooltip } from "@/components/Forecast/echarts";

// chartOption is provided by each consuming component's computed
type WithChartOption = { chartOption: Record<string, unknown> };

// Shared echarts lifecycle: init on a `chartEl` template ref (lazily re-checked
// so v-if-gated charts work), option updates via deep watch, window resizing,
// touch tooltips, disposal. Override `applyChartOption`, `onChartInit`,
// `onTouchTooltipReset` or `resize` for custom behavior.
export default defineComponent({
  data(): { chart: echarts.ECharts | null } {
    return { chart: null };
  },
  watch: {
    chartOption: {
      handler() {
        this.$nextTick(() => {
          this.initChart();
          this.applyChartOption();
        });
      },
      deep: true,
    },
  },
  mounted() {
    this.initChart();
    window.addEventListener("resize", this.resize);
  },
  beforeUnmount() {
    window.removeEventListener("resize", this.resize);
    this.chart?.dispose();
    this.chart = null;
  },
  methods: {
    initChart() {
      const el = this.$refs["chartEl"] as HTMLElement | undefined;
      if (this.chart) {
        if (el && this.chart.getDom() === el) return;
        // element removed or replaced by v-if
        this.chart.dispose();
        this.chart = null;
      }
      if (!el) return;
      this.chart = markRaw(echarts.init(el));
      this.chart.setOption((this as unknown as WithChartOption).chartOption);
      registerTouchTooltip(this.chart, el, () => this.onTouchTooltipReset());
      this.onChartInit();
    },
    applyChartOption() {
      this.chart?.setOption((this as unknown as WithChartOption).chartOption);
    },
    onChartInit() {},
    onTouchTooltipReset() {},
    resize() {
      this.chart?.resize();
    },
  },
});
