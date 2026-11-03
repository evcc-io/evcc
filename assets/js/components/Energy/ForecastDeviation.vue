<template>
	<div ref="chartEl" class="forecast-deviation" data-testid="forecast-deviation"></div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import echartsChart from "@/mixins/echartsChart";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import colors from "@/colors";
import { slotWatts } from "./slots";
import { tooltipStyle, tooltipTable } from "../Forecast/echarts";

// forecast accuracy per bucket: axis-less segments from the forecast to what was
// produced, with a tooltip named by `labels` that lists both and the deviation
export default defineComponent({
	name: "ForecastDeviation",
	mixins: [echartsChart, formatter],
	props: {
		// per bucket the forecast and what was produced, null without a forecast
		spans: { type: Array as PropType<([number, number] | null)[]>, default: () => [] },
		color: { type: String, default: "" },
		labels: { type: Array as PropType<string[]>, default: () => [] },
		// 15 minute buckets read as average power
		power: Boolean,
	},
	computed: {
		chartOption(): Record<string, unknown> {
			// each segment as its lower end and length
			const segments = this.spans.map((sp) =>
				sp ? [Math.min(sp[0], sp[1]), Math.abs(sp[1] - sp[0])] : [null, null]
			);
			const shared = { type: "bar", stack: "span", barCategoryGap: "0%", silent: true };
			return {
				animation: false,
				grid: { left: 0, right: 0, top: 2, bottom: 2 },
				tooltip: {
					trigger: "axis",
					axisPointer: { type: "shadow", shadowStyle: { color: "transparent" } },
					...tooltipStyle(colors.text || ""),
					formatter: (params: { dataIndex: number }[]) => {
						const idx = params[0]?.dataIndex ?? 0;
						const span = this.spans[idx];
						if (!span) return "";
						const fmt = (v: number, signed = false) =>
							this.power
								? this.fmtW(slotWatts(v), POWER_UNIT.AUTO, true, undefined, signed)
								: this.fmtKWh(v, signed);
						// forecast first, the deviation signed from the forecast's side
						return tooltipTable(this.labels[idx] ?? "", [
							{
								name: this.$t("energy.group.forecast"),
								values: [fmt(span[0])],
							},
							{ name: this.$t("energy.production.produced"), values: [fmt(span[1])] },
							{
								name: this.$t("energy.production.deviation"),
								values: [fmt(span[0] - span[1], true)],
							},
						]);
					},
				},
				xAxis: {
					type: "category",
					data: this.spans.map((_, i) => i),
					axisLine: { onZero: true, lineStyle: { color: colors.border || "", width: 1 } },
					axisTick: { show: false },
					axisLabel: { show: false },
				},
				yAxis: { type: "value", show: false, min: 0 },
				series: [
					{
						...shared,
						data: segments.map((s) => s[0]),
						itemStyle: { color: "transparent" },
					},
					// a hairline where forecast and actual agree, so the row reads as covered
					{
						...shared,
						data: segments.map((s) => s[1]),
						itemStyle: { color: this.color },
						barMinHeight: 1,
					},
				],
			};
		},
	},
});
</script>

<style scoped>
.forecast-deviation {
	width: 100%;
	height: 48px;
}
</style>
