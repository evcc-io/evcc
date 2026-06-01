<template>
	<div ref="scrollEl" class="forecast-chart-scroll scroll-overlay-fix" @scroll="onScroll">
		<div ref="chartEl" :style="{ height: '200px', width: chartWidth + 'px' }"></div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import {
	FONT_FAMILY,
	markPointLabel,
	tooltipStyle,
	forecastGrid,
	forecastXAxes,
	forecastYAxis,
} from "./echarts";
import colors, { lighterColor } from "@/colors";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import chartMixin from "./chartMixin";
import { highestSlotIndexByDay } from "@/utils/forecast";
import type { SolarDetails, TimeseriesEntry } from "./types";

export default defineComponent({
	name: "SolarChart",
	mixins: [formatter, chartMixin],
	props: {
		solar: { type: Object as PropType<SolarDetails> },
		rawSolar: { type: Object as PropType<SolarDetails> },
	},
	computed: {
		entries(): TimeseriesEntry[] {
			return (this.solar?.timeseries || []).filter(
				(e) => new Date(e.ts) >= this.startDate && new Date(e.ts) <= this.endDate
			);
		},
		combinedMax(): number {
			let max = 0;
			for (const src of [this.solar, this.rawSolar]) {
				for (const e of src?.timeseries || []) {
					if (e.val > max) max = e.val;
				}
			}
			return max;
		},
		markPoints(): { coord: [string, number]; value: string }[] {
			const points: { coord: [string, number]; value: string }[] = [];
			const days = [
				{ energy: this.solar?.today?.energy, day: 0 },
				{ energy: this.solar?.tomorrow?.energy, day: 1 },
				{ energy: this.solar?.dayAfterTomorrow?.energy, day: 2 },
			];
			for (const { energy, day } of days) {
				const idx = highestSlotIndexByDay(this.entries, day);
				if (idx >= 0 && this.entries[idx] && energy) {
					const entry = this.entries[idx]!;
					points.push({
						coord: [entry.ts, entry.val],
						value: this.fmtWh(energy, POWER_UNIT.AUTO),
					});
				}
			}
			return points;
		},
		chartOption(): Record<string, unknown> {
			const selfColor = colors.self || "";
			const data = this.entries.map((e) => [e.ts, e.val]);

			return {
				animationDuration: 0,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: forecastGrid(),
				tooltip: {
					trigger: "axis",
					axisPointer: {
						type: "line",
						snap: true,
						snapThreshold: 50,
						lineStyle: { color: "transparent" },
					},
					...tooltipStyle(selfColor, () => this.chart),
					formatter: (params: { value: [string, number] }[]) => {
						const p = params[0];
						if (!p) return "";
						const d = new Date(p.value[0]);
						const time = `${this.weekdayShort(d)} ${this.fmtHourMinute(d)}`;
						return `${time}<br/>${this.fmtW(p.value[1], POWER_UNIT.AUTO)}`;
					},
				},
				xAxis: forecastXAxes(this.startDate, this.endDate, this.weekdayShort),
				yAxis: forecastYAxis({
					max: (value: { max: number }) => {
						const m = Math.max(value.max, this.combinedMax);
						const step = Math.pow(10, Math.floor(Math.log10(m || 1)));
						return Math.ceil(m / step) * step;
					},
					splitNumber: 2,
					axisLabel: {
						color: colors.muted,
						formatter: (value: number) => this.fmtW(value, POWER_UNIT.KW, false, 0),
					},
				}),
				series: [
					{
						type: "line",
						data,
						smooth: true,
						symbol: "circle",
						symbolSize: 6,
						showSymbol: false,
						lineStyle: { color: selfColor, width: 3 },
						areaStyle: { color: lighterColor(selfColor) },
						emphasis: {
							disabled: false,
							scale: false,
							lineStyle: { color: selfColor, width: 3 },
							areaStyle: { color: lighterColor(selfColor) },
							itemStyle: { color: selfColor, borderColor: selfColor, borderWidth: 2 },
						},
						markPoint: markPointLabel(
							selfColor,
							this.tooltipVisible ? [] : this.markPoints,
							this.startDate,
							this.endDate
						),
					},
				],
			};
		},
	},
});
</script>
