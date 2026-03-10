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
import type { CURRENCY } from "@/types/evcc";
import type { ForecastSlot } from "./types";

export default defineComponent({
	name: "PriceChart",
	mixins: [formatter],
	props: {
		grid: { type: Array as PropType<ForecastSlot[]>, required: true },
		currency: { type: String as PropType<CURRENCY> },
		chartWidth: { type: Number, required: true },
		endDate: { type: Date, required: true },
		scrollLeft: { type: Number, default: 0 },
		zoom: { type: Boolean, default: false },
	},
	emits: ["scroll"],
	data(): {
		chart: echarts.ECharts | null;
		startDate: Date;
		tooltipVisible: boolean; hoveredIndex: number;
	} {
		return {
			chart: null,
			startDate: new Date(),
			tooltipVisible: false, hoveredIndex: -1 as number,
		};
	},
	computed: {
		slots(): ForecastSlot[] {
			return this.filterSlots(this.grid);
		},
		nextMidnight(): Date {
			const d = new Date(this.startDate);
			d.setDate(d.getDate() + 1);
			d.setHours(0, 0, 0, 0);
			return d;
		},
		markPoints(): { coord: [string, number]; value: string }[] {
			const slots = this.slots;
			if (!slots.length) return [];
			const minIdx = this.minIndex(slots);
			const maxIdx = this.maxIndex(slots);
			const points: { coord: [string, number]; value: string }[] = [];
			if (slots[minIdx]) {
				points.push({
					coord: [this.clampStart(slots[minIdx]!.start), slots[minIdx]!.value],
					value: this.fmtPricePerKWh(slots[minIdx]!.value, this.currency, true, true),
				});
			}
			if (maxIdx !== minIdx && slots[maxIdx]) {
				points.push({
					coord: [this.clampStart(slots[maxIdx]!.start), slots[maxIdx]!.value],
					value: this.fmtPricePerKWh(slots[maxIdx]!.value, this.currency, true, true),
				});
			}
			return points;
		},
		yAxisConfig(): Record<string, unknown> {
			const values = this.slots.map((s) => s.value);
			const dataMin = Math.min(...values);
			const dataMax = Math.max(...values);
			// compute nice interval from full 0-based range
			const fullMin = Math.min(0, dataMin);
			const fullRange = dataMax - fullMin || 1;
			const rawInterval = fullRange / 5;
			const magnitude = Math.pow(10, Math.floor(Math.log10(rawInterval)));
			const nice = [1, 2, 2.5, 5, 10].find((n) => n * magnitude >= rawInterval) || 10;
			const interval = nice * magnitude;

			if (this.zoom) {
				// tight range, keep same interval so ticks are a subset
				return {
					min: Math.floor(dataMin / interval) * interval,
					max: Math.ceil(dataMax / interval) * interval,
					interval,
				};
			}
			return {
				min: fullMin,
				max: Math.ceil(dataMax / interval) * interval,
				interval,
			};
		},
		chartOption(): Record<string, unknown> {
			const priceColor = colors.price || "#FF922EFF";

			// eslint-disable-next-line @typescript-eslint/no-this-alias
			const vThis = this;
			return {
				animationDuration: 0,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: { top: 36, right: 16, bottom: 4, left: 40, borderWidth: 0 },
				tooltip: {
					trigger: "axis",
					axisPointer: { type: "line", snap: true, lineStyle: { color: "transparent" } },
					...tooltipStyle(priceColor, () => this.chart),
					formatter(params: { value: [string, number] }[]) {
						const p = params[0];
						if (!p) return "";
						const d = new Date(p.value[0]);
						const time = `${vThis.weekdayShort(d)} ${vThis.fmtHourMinute(d)}`;
						return `${time}<br/>${vThis.fmtPricePerKWh(p.value[1], vThis.currency, false, true)}`;
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
					...this.yAxisConfig,
					axisLine: { show: false },
					axisTick: { show: false },
					axisLabel: {
						color: colors.muted,
						formatter: (value: number) => {
							const v =
								this.currency && this.energyPriceSubunit(this.currency)
									? value * 100
									: value;
							return `${Math.round(v)}`;
						},
					},
					splitLine: {
						showMinLine: false,
						showMaxLine: false,
						lineStyle: { color: colors.border || "#eee" },
					},
				},
				series: [
					{
						type: "bar", cursor: "default",
						data: this.slots.map((s, i) => ({
							value: [this.clampStart(s.start), s.value],
							itemStyle: this.hoveredIndex >= 0 && i !== this.hoveredIndex
								? { opacity: 0.33 }
								: undefined,
						})),
						barMaxWidth: 4,
						barMinWidth: 4,
						itemStyle: {
							color: priceColor,
							borderRadius: [2, 2, 0, 0],
						},
						emphasis: { disabled: true },
						markPoint: markPointLabel(
							priceColor,
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
		this.chart.on("showTip", (params: unknown) => { const p = params as { dataIndex?: number };
			this.tooltipVisible = true;
			if (p.dataIndex != null) this.hoveredIndex = p.dataIndex;
		});
		this.chart.on("hideTip", () => {
			this.tooltipVisible = false;
			this.hoveredIndex = -1;
		});
		this.chart.getZr().on("mouseout", () => {
			this.tooltipVisible = false;
			this.hoveredIndex = -1;
		});
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
			if (!Array.isArray(slots)) return [];
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
