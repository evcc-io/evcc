<template>
	<div ref="scrollEl" class="forecast-chart-scroll" @scroll="onScroll">
		<div ref="chartEl" :style="{ height: '200px', width: chartWidth + 'px' }"></div>
	</div>
</template>

<script lang="ts">
import { defineComponent, markRaw, type PropType } from "vue";
import { echarts, FONT_FAMILY, markPointLabel, tooltipStyle } from "./echarts";
import colors, { lighterColor } from "@/colors";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import { highestSlotIndexByDay } from "@/utils/forecast";
import type { SolarDetails, TimeseriesEntry } from "./types";

export default defineComponent({
	name: "SolarChart",
	mixins: [formatter],
	props: {
		solar: { type: Object as PropType<SolarDetails> },
		rawSolar: { type: Object as PropType<SolarDetails> },
		chartWidth: { type: Number, required: true },
		endDate: { type: Date, required: true },
		scrollLeft: { type: Number, default: 0 },
	},
	emits: ["scroll"],
	data(): {
		chart: echarts.ECharts | null;
		startDate: Date; tooltipVisible: boolean;
	} {
		return {
			chart: null, tooltipVisible: false,
			startDate: new Date(),
		};
	},
	computed: {
		nextMidnight(): Date {
			const d = new Date(this.startDate);
			d.setDate(d.getDate() + 1);
			d.setHours(0, 0, 0, 0);
			return d;
		},
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
			const selfColor = colors.self || "#0FDE41FF";
			const data = this.entries.map((e) => [e.ts, e.val]);

			return {
				animationDuration: 0,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: { top: 36, right: 16, bottom: 4, left: 40, borderWidth: 0 },
				tooltip: {
					trigger: "axis",
					axisPointer: { type: "line", snap: true, snapThreshold: 50, lineStyle: { color: "transparent" } },
					...tooltipStyle(selfColor, () => this.chart),
					formatter: (params: { value: [string, number] }[]) => {
						const p = params[0];
						if (!p) return "";
						const d = new Date(p.value[0]);
						const time = `${this.weekdayShort(d)} ${this.fmtHourMinute(d)}`;
						return `${time}<br/>${this.fmtW(p.value[1], POWER_UNIT.AUTO)}`;
					},
				},
				xAxis: [
					{
						type: "time",
						min: this.startDate,
						max: this.endDate,
						minInterval: 3600 * 1000,
						maxInterval: 3600 * 1000,
						axisLabel: {
							color: colors.muted,
							formatter: (value: number) => {
								const date = new Date(value);
								const h = date.getHours();
								if (h === 0) return `${h}\n${this.weekdayShort(date)}`;
								return `${h}`;
							},
						},
						splitLine: { show: false },
						axisLine: { show: false },
						axisTick: { show: false },
					},
					{
						type: "time",
						position: "bottom",
						min: this.startDate,
						max: this.endDate,
						minInterval: 24 * 3600 * 1000,
						maxInterval: 24 * 3600 * 1000,
						axisLabel: { show: false },
						axisLine: { show: false },
						axisTick: { show: false },
						splitLine: {
							show: true,
							showMinLine: false,
							showMaxLine: false,
							lineStyle: { color: colors.border || "#eee", type: "dashed" },
						},
					},
				],
				yAxis: {
					type: "value",
					min: 0,
					axisLine: { show: false },
					axisTick: { show: false },
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
					splitLine: {
						showMinLine: false,
						showMaxLine: false,
						lineStyle: { color: colors.border || "#eee" },
					},
				},
				series: [
					{
						type: "line",
						data,
						smooth: 0.05,
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
	watch: {
		chartOption: {
			handler() {
				this.chart?.setOption(this.chartOption);
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
		const el = this.$refs["chartEl"] as HTMLElement;
		this.chart = markRaw(echarts.init(el));
		this.chart.setOption(this.chartOption);
		this.chart.on("showTip", () => { this.tooltipVisible = true; });
		this.chart.on("hideTip", () => { this.tooltipVisible = false; });
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
	},
});
</script>

<style scoped>
.forecast-chart-scroll {
	overflow-x: auto;
	margin-bottom: 1.5rem;
	padding-bottom: 4px;
}
</style>
