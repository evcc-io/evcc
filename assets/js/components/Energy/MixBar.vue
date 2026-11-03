<template>
	<div ref="chartEl" class="mix"></div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import echartsChart from "@/mixins/echartsChart";
import colors from "@/colors";
import { tooltipStyle, tooltipTable, type TooltipRow } from "../Forecast/echarts";

// proportional fill bar, segments grow with their share of the sum and keep a small
// gap between them. One text line tall with the bar centered, so it aligns with
// sublines beside it and the whole line takes the touch
const GLIDE = { duration: 500, easing: "exponentialOut" };

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
			// a part thinner than the gap would only eat the pill's rounded end
			const drawn = this.segments.filter((s) => s.value >= total / 100 && s.value > 0);
			// every segment keeps its place in the stack, undrawn ones at zero width, so
			// the order survives updates. A sliver of nothing sits before each drawn
			// segment but the first, so neighbours read as separate parts
			const parts = this.segments.flatMap((s, i) => {
				const shown = drawn.includes(s);
				const first = s === drawn[0] ? 3 : 0;
				const last = s === drawn.at(-1) ? 3 : 0;
				return [
					{ value: shown && !first ? total / 100 : 0, color: "transparent", radius: 0 },
					// pill ends on the outer drawn segments
					{
						value: shown ? s.value : 0,
						color: s.color,
						radius: [first, last, last, first],
					},
				].slice(i ? 0 : 1);
			});
			return {
				// values glide in step with the number above the bar (AnimatedNumber)
				animationDuration: GLIDE.duration,
				animationEasing: GLIDE.easing,
				animationDurationUpdate: GLIDE.duration,
				animationEasingUpdate: GLIDE.easing,
				grid: { left: 0, right: 0, top: 0, bottom: 0 },
				// the stack fills the width, echarts would otherwise round the axis up
				xAxis: {
					type: "value",
					show: false,
					max: parts.reduce((acc, s) => acc + s.value, 0) || 1,
				},
				yAxis: { type: "category", show: false, data: [""] },
				tooltip: {
					show: this.tooltip.length > 0,
					trigger: "axis",
					axisPointer: { type: "none" },
					...tooltipStyle(colors.text || ""),
					formatter: () => tooltipTable("", this.tooltip),
				},
				series: parts.map((s) => ({
					type: "bar",
					stack: "mix",
					data: [s.value],
					itemStyle: { color: s.color, borderRadius: s.radius },
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
