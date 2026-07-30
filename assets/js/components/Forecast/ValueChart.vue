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
	tooltipTable,
	forecastGrid,
	forecastXAxes,
	forecastYAxis,
	clampStart,
	filterForecastSlots,
	minSlotIndex,
	maxSlotIndex,
} from "./echarts";
import colors from "@/colors";
import formatter from "@/mixins/formatter";
import chartMixin from "./chartMixin";
import type { ForecastSlot } from "./types";

export type ValueChartType = "co2" | "temperature";

export default defineComponent({
	name: "ValueChart",
	mixins: [formatter, chartMixin],
	props: {
		type: { type: String as PropType<ValueChartType>, required: true },
		rates: { type: Array as PropType<ForecastSlot[]>, required: true },
	},
	computed: {
		color(): string {
			return (this.type === "co2" ? colors.co2 : colors.temperature) || "";
		},
		slots(): ForecastSlot[] {
			return filterForecastSlots(this.rates, this.startDate, this.endDate);
		},
		yMin(): number {
			// co2 is never negative, temperature may be
			if (this.type === "co2" || !this.slots.length) return 0;
			return Math.min(0, Math.floor(Math.min(...this.slots.map((s) => s.value))));
		},
		markPoints(): {
			coord: [string, number];
			value: string;
			label?: Record<string, unknown>;
		}[] {
			const slots = this.slots;
			if (!slots.length) return [];
			const minIdx = minSlotIndex(slots);
			const maxIdx = maxSlotIndex(slots);
			const points: {
				coord: [string, number];
				value: string;
				label?: Record<string, unknown>;
			}[] = [];
			if (slots[minIdx]) {
				points.push({
					coord: [clampStart(slots[minIdx]!.start, this.startDate), slots[minIdx]!.value],
					value: this.fmtShort(slots[minIdx]!.value),
					label: { position: "bottom", offset: [0, 2] },
				});
			}
			if (maxIdx !== minIdx && slots[maxIdx]) {
				points.push({
					coord: [clampStart(slots[maxIdx]!.start, this.startDate), slots[maxIdx]!.value],
					value: this.fmtShort(slots[maxIdx]!.value),
				});
			}
			return points;
		},
		chartOption(): Record<string, unknown> {
			const color = this.color;

			// eslint-disable-next-line @typescript-eslint/no-this-alias
			const vThis = this;
			return {
				animationDuration: 0,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: forecastGrid(),
				tooltip: {
					trigger: "axis",
					axisPointer: { type: "line", snap: true, lineStyle: { color: "transparent" } },
					...tooltipStyle(color, () => this.chart),
					formatter(params: { value: [string, number] }[]) {
						const p = params[0];
						if (!p) return "";
						const d = new Date(p.value[0]);
						const time = `${vThis.weekdayShort(d)} ${vThis.fmtHourMinute(d)}`;
						return tooltipTable(time, [{ values: [vThis.fmtLong(p.value[1])] }]);
					},
				},
				xAxis: forecastXAxes(
					this.startDate,
					this.endDate,
					this.hourShort,
					this.weekdayShort
				),
				yAxis: forecastYAxis({
					min: this.yMin,
					splitNumber: 2,
					axisLabel: {
						color: colors.muted,
						formatter: (value: number) => `${Math.round(value)}`,
					},
				}),
				series: [
					{
						type: "line",
						data: this.slots.map((s) => [s.start, s.value]),
						smooth: true,
						symbol: "circle",
						symbolSize: 6,
						showSymbol: false,
						lineStyle: { color, width: 3 },
						emphasis: {
							disabled: false,
							scale: false,
							itemStyle: { color, borderColor: color, borderWidth: 2 },
						},
						markPoint: markPointLabel(
							color,
							this.tooltipVisible ? [] : this.markPoints,
							this.startDate,
							this.endDate
						),
					},
				],
			};
		},
	},
	methods: {
		fmtShort(value: number): string {
			return this.type === "co2" ? this.fmtGrams(value) : this.fmtTemperature(value);
		},
		fmtLong(value: number): string {
			return this.type === "co2" ? this.fmtCo2Medium(value) : this.fmtTemperature(value);
		},
	},
});
</script>
