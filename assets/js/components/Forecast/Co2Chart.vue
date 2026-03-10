<template>
	<div ref="scrollEl" class="forecast-chart-scroll" @scroll="onScroll">
		<div ref="chartEl" :style="{ height: '200px', width: chartWidth + 'px' }"></div>
	</div>
</template>

<script lang="ts">
import { defineComponent, markRaw, type PropType } from "vue";
import { echarts, FONT_FAMILY, markPointLabel, tooltipStyle } from "./echarts";
import colors from "@/colors";
import formatter from "@/mixins/formatter";
import type { ForecastSlot } from "./types";

export default defineComponent({
	name: "Co2Chart",
	mixins: [formatter],
	props: {
		co2: { type: Array as PropType<ForecastSlot[]>, required: true },
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
		slots(): ForecastSlot[] {
			return this.filterSlots(this.co2);
		},
		nextMidnight(): Date {
			const d = new Date(this.startDate);
			d.setDate(d.getDate() + 1);
			d.setHours(0, 0, 0, 0);
			return d;
		},
		markPoints(): {
			coord: [string, number];
			value: string;
			label?: Record<string, unknown>;
		}[] {
			const slots = this.slots;
			if (!slots.length) return [];
			const minIdx = this.minIndex(slots);
			const maxIdx = this.maxIndex(slots);
			const points: {
				coord: [string, number];
				value: string;
				label?: Record<string, unknown>;
			}[] = [];
			if (slots[minIdx]) {
				points.push({
					coord: [this.clampStart(slots[minIdx]!.start), slots[minIdx]!.value],
					value: this.fmtGrams(slots[minIdx]!.value),
					label: { position: "bottom", offset: [0, 2] },
				});
			}
			if (maxIdx !== minIdx && slots[maxIdx]) {
				points.push({
					coord: [this.clampStart(slots[maxIdx]!.start), slots[maxIdx]!.value],
					value: this.fmtGrams(slots[maxIdx]!.value),
				});
			}
			return points;
		},
		chartOption(): Record<string, unknown> {
			const co2Color = colors.co2 || "#03C1EFFF";

			// eslint-disable-next-line @typescript-eslint/no-this-alias
			const vThis = this;
			return {
				animationDuration: 0,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: { top: 36, right: 16, bottom: 4, left: 40, borderWidth: 0 },
				tooltip: {
					trigger: "axis",
					axisPointer: { type: "line", snap: true, lineStyle: { color: "transparent" } },
					...tooltipStyle(co2Color, () => this.chart),
					formatter(params: { value: [string, number] }[]) {
						const p = params[0];
						if (!p) return "";
						const d = new Date(p.value[0]);
						const time = `${vThis.weekdayShort(d)} ${vThis.fmtHourMinute(d)}`;
						return `${time}<br/>${vThis.fmtCo2Medium(p.value[1])}`;
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
					splitNumber: 2,
					axisLine: { show: false },
					axisTick: { show: false },
					axisLabel: {
						color: colors.muted,
						formatter: (value: number) => `${Math.round(value)}`,
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
						data: this.slots.map((s) => [s.start, s.value]),
						smooth: 0.05,
						symbol: "circle",
						symbolSize: 6,
						showSymbol: false,
						lineStyle: { color: co2Color, width: 3 },
						emphasis: {
							disabled: false,
							scale: false,
							itemStyle: { color: co2Color, borderColor: co2Color, borderWidth: 2 },
						},
						markPoint: markPointLabel(
							co2Color,
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
		clampStart(ts: string): string {
			return new Date(ts) < this.startDate ? this.startDate.toISOString() : ts;
		},
		filterSlots(slots: ForecastSlot[]): ForecastSlot[] {
			return slots.filter(
				(s) => new Date(s.end) >= this.startDate && new Date(s.start) <= this.endDate
			);
		},
		maxIndex(slots: ForecastSlot[]): number {
			return slots.reduce((max, s, i) => (s.value > (slots[max]?.value || 0) ? i : max), 0);
		},
		minIndex(slots: ForecastSlot[]): number {
			return slots.reduce(
				(min, s, i) => (s.value < (slots[min]?.value || Infinity) ? i : min),
				0
			);
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
