<template>
	<div ref="scrollEl" class="forecast-chart-scroll scroll-overlay-fix" @scroll="onScroll">
		<div ref="chartEl" :style="{ height: '200px', width: chartWidth + 'px' }"></div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import {
	echarts,
	FONT_FAMILY,
	markPointLabel,
	tooltipStyle,
	forecastGrid,
	forecastXAxes,
	forecastYAxis,
	clampStart,
	filterForecastSlots,
	minSlotIndex,
	maxSlotIndex,
} from "./echarts";
import colors, { lighterColor } from "@/colors";
import formatter from "@/mixins/formatter";
import chartMixin from "./chartMixin";
import type { CURRENCY } from "@/types/evcc";
import type { ForecastSlot } from "./types";

export default defineComponent({
	name: "PriceChart",
	mixins: [formatter, chartMixin],
	props: {
		grid: { type: Array as PropType<ForecastSlot[]>, required: true },
		feedin: { type: Array as PropType<ForecastSlot[]> },
		currency: { type: String as PropType<CURRENCY> },
		zoom: { type: Boolean, default: false },
	},
	computed: {
		slots(): ForecastSlot[] {
			return filterForecastSlots(this.grid, this.startDate, this.endDate);
		},
		feedinSlots(): ForecastSlot[] {
			return this.feedin
				? filterForecastSlots(this.feedin, this.startDate, this.endDate)
				: [];
		},
		markPoints(): { coord: [string, number]; value: string }[] {
			const slots = this.slots;
			if (!slots.length) return [];
			const minIdx = minSlotIndex(slots);
			const maxIdx = maxSlotIndex(slots);
			const points: { coord: [string, number]; value: string }[] = [];
			if (slots[minIdx]) {
				points.push({
					coord: [clampStart(slots[minIdx]!.start, this.startDate), slots[minIdx]!.value],
					value: this.fmtPricePerKWh(slots[minIdx]!.value, this.currency, true, true),
				});
			}
			if (maxIdx !== minIdx && slots[maxIdx]) {
				points.push({
					coord: [clampStart(slots[maxIdx]!.start, this.startDate), slots[maxIdx]!.value],
					value: this.fmtPricePerKWh(slots[maxIdx]!.value, this.currency, true, true),
				});
			}
			return points;
		},
		yAxisConfig(): Record<string, unknown> {
			const values = [
				...this.slots.map((s) => s.value),
				...this.feedinSlots.map((s) => s.value),
			];
			const dataMin = Math.min(...values);
			const dataMax = Math.max(...values);
			const rangeMin = this.zoom ? dataMin : Math.min(0, dataMin);
			const rangeMax = Math.max(0, dataMax);
			const range = rangeMax - rangeMin || 1;
			const rawInterval = range / 5;
			const magnitude = Math.pow(10, Math.floor(Math.log10(rawInterval)));
			const nice = [1, 2, 2.5, 5, 10].find((n) => n * magnitude >= rawInterval) || 10;
			const interval = nice * magnitude;

			return {
				min: Math.floor(rangeMin / interval) * interval,
				max: Math.ceil(rangeMax / interval) * interval,
				interval,
			};
		},
		chartOption(): Record<string, unknown> {
			const priceColor = colors.price || "";
			const exportColor = colors.export || "";

			// eslint-disable-next-line @typescript-eslint/no-this-alias
			const vThis = this;
			return {
				animationDuration: 0,
				animationDurationUpdate: 300,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: forecastGrid(),
				tooltip: {
					trigger: "axis",
					axisPointer: { type: "line", snap: true, lineStyle: { color: "transparent" } },
					...tooltipStyle(priceColor, () => this.chart),
					formatter(params: { value: [string, number]; seriesIndex: number }[]) {
						const p = params[0];
						if (!p) return "";
						const d = new Date(p.value[0]);
						const time = `${vThis.weekdayShort(d)} ${vThis.fmtHourMinute(d)}`;
						const lines = [time];
						const showLabels = params.length > 1;
						const labels = [
							vThis.$t("main.energyflow.gridImport"),
							vThis.$t("main.energyflow.pvExport"),
						];
						for (const s of params) {
							const price = vThis.fmtPricePerKWh(
								s.value[1],
								vThis.currency,
								true,
								true
							);
							const label = showLabels ? `${labels[s.seriesIndex]}: ` : "";
							lines.push(`${label}${price}`);
						}
						return lines.join("<br/>");
					},
				},
				xAxis: forecastXAxes(this.startDate, this.endDate, this.weekdayShort),
				yAxis: forecastYAxis({
					...this.yAxisConfig,
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
				}),
				series: [
					this.priceSeries(this.slots, priceColor, this.markPoints),
					this.priceSeries(this.feedinSlots, exportColor),
				],
			};
		},
	},
	methods: {
		priceSeries(
			slots: ForecastSlot[],
			color: string,
			points?: { coord: [string, number]; value: string }[]
		): Record<string, unknown> {
			const avg = slots.length ? slots.reduce((a, s) => a + s.value, 0) / slots.length : 0;
			const gradientDown = avg >= 0;
			return {
				type: "line",
				step: "start",
				cursor: "default",
				showSymbol: false,
				data: slots.map((s) => ({
					value: [clampStart(s.start, this.startDate), s.value],
				})),
				lineStyle: { color, width: 2 },
				areaStyle: {
					color: new echarts.graphic.LinearGradient(
						0,
						gradientDown ? 0 : 1,
						0,
						gradientDown ? 1 : 0,
						[
							{ offset: 0, color: lighterColor(color) || color },
							{ offset: 0.75, color: color + "00" },
							{ offset: 1, color: color + "00" },
						]
					),
				},
				itemStyle: { color },
				emphasis: { disabled: true },
				...(points
					? {
							markPoint: markPointLabel(
								color,
								this.tooltipVisible ? [] : points,
								this.startDate,
								this.endDate
							),
						}
					: {}),
			};
		},
	},
});
</script>
