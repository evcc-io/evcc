<template>
	<div ref="chartEl" class="mix"></div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import echartsChart from "@/mixins/echartsChart";
import colors from "@/colors";
import { tooltipStyle, tooltipTable, type TooltipRow } from "../Forecast/echarts";

// proportional fill bar, segments grow with their share of the sum. One text line
// tall with the bar centered, so it aligns with sublines beside it and the whole
// line takes the touch
export default defineComponent({
	name: "MixBar",
	mixins: [echartsChart],
	props: {
		segments: { type: Array as PropType<{ value: number; color: string }[]>, required: true },
		// none for a bar without a tooltip
		tooltip: { type: Array as PropType<TooltipRow[]>, default: () => [] },
	},
	computed: {
		chartOption(): Record<string, unknown> {
			const total = this.segments.reduce((acc, s) => acc + s.value, 0);
			const drawn = this.segments.filter((s) => s.value > 0);
			// pill ends on the outer drawn segments
			const radius = (s: { value: number }) => {
				const first = s === drawn[0] ? 3 : 0;
				const last = s === drawn.at(-1) ? 3 : 0;
				return [first, last, last, first];
			};
			return {
				grid: { left: 0, right: 0, top: 0, bottom: 0 },
				// the stack fills the width, echarts would otherwise round the axis up
				xAxis: { type: "value", show: false, max: total || 1 },
				yAxis: { type: "category", show: false, data: [""] },
				tooltip: {
					show: this.tooltip.length > 0,
					trigger: "axis",
					axisPointer: { type: "none" },
					...tooltipStyle(colors.text || ""),
					formatter: () => tooltipTable("", this.tooltip),
				},
				series: this.segments.map((s) => ({
					type: "bar",
					stack: "mix",
					data: [s.value],
					itemStyle: { color: s.color, borderRadius: radius(s) },
					barWidth: 6,
					silent: true,
				})),
			};
		},
	},
});
</script>

<style scoped>
.mix {
	height: 1.5em;
}
</style>
