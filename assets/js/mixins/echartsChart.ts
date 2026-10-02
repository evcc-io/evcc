import { defineComponent, markRaw } from "vue";
import { echarts, registerTouchTooltip } from "@/components/Forecast/echarts";

// chartOption is provided by each consuming component's computed
type WithChartOption = { chartOption: Record<string, unknown> };

// Shared echarts lifecycle: init on a `chartEl` template ref (lazily re-checked
// so v-if-gated charts work), option updates via deep watch, resizing with the
// element (flex and grid lay tiles out after the chart exists), touch tooltips, disposal. Override `applyChartOption`, `onChartInit`,
// `onTouchTooltipReset`, `onChartTap` or `resize` for custom behavior.
export default defineComponent({
  data(): { chart: echarts.ECharts | null; observer: ResizeObserver | null } {
    return { chart: null, observer: null };
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
  },
  beforeUnmount() {
    this.observer?.disconnect();
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
      this.observer?.disconnect();
      this.observer = new ResizeObserver(() => this.resize());
      this.observer.observe(el);
      // hover states jump instead of fading, keeps the dot in sync with the tooltip
      this.chart.setOption({ stateAnimation: { duration: 0 } });
      this.chart.setOption((this as unknown as WithChartOption).chartOption);
      registerTouchTooltip(
        this.chart,
        el,
        () => this.onTouchTooltipReset(),
        (x, y) => (this.onChartTap as (x: number, y: number) => void)(x, y)
      );
      this.onChartInit();
    },
    applyChartOption() {
      this.chart?.setOption((this as unknown as WithChartOption).chartOption);
    },
    onChartInit() {},
    onTouchTooltipReset() {},
    // tap on touch devices, pixel position relative to the chart element
    onChartTap() {},
    resize() {
      this.chart?.resize();
    },
  },
});
