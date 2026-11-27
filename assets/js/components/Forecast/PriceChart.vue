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
	tooltipTable,
	forecastGrid,
	forecastXAxes,
	forecastYAxis,
	clampStart,
	filterForecastSlots,
	minSlotIndex,
	maxSlotIndex,
	hoverDot,
	lineDefaults,
	hatchPattern,
} from "./echarts";
import colors, { lighterColor } from "@/colors";
import formatter from "@/mixins/formatter";
import chartMixin from "./chartMixin";
import type { CURRENCY, UiForecastSlot } from "@/types/evcc";

export default defineComponent({
	name: "PriceChart",
	mixins: [formatter, chartMixin],
	props: {
		grid: { type: Array as PropType<UiForecastSlot[]>, required: true },
		feedin: { type: Array as PropType<UiForecastSlot[]> },
		currency: { type: String as PropType<CURRENCY> },
		zoom: { type: Boolean, default: false },
	},
	computed: {
		slots(): UiForecastSlot[] {
			return filterForecastSlots(this.grid, this.startDate, this.endDate);
		},
		feedinSlots(): UiForecastSlot[] {
			return this.feedin
				? filterForecastSlots(this.feedin, this.startDate, this.endDate)
				: [];
		},
		markPoints(): { coord: [number, number]; value: string }[] {
			const slots = this.slots;
			if (!slots.length) return [];
			const minIdx = minSlotIndex(slots);
			const maxIdx = maxSlotIndex(slots);
			const points: { coord: [number, number]; value: string }[] = [];
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

			// oxlint-disable-next-line typescript/no-this-alias
			const vThis = this;
			return {
				animationDuration: 0,
				animationDurationUpdate: 300,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: forecastGrid(),
				tooltip: {
					trigger: "axis",
					axisPointer: { type: "line", snap: true, lineStyle: { color: "transparent" } },
					...tooltipStyle(priceColor),
					formatter(
						params: {
							value: [string, number];
							seriesName: string;
							data?: { closing?: boolean; estimate?: boolean };
						}[]
					) {
						const points = params.filter((s) => !s.data?.closing);
						const p = points[0];
						if (!p) return "";
						const d = new Date(p.value[0]);
						const time = `${vThis.weekdayShort(d)} ${vThis.fmtHourMinute(d)}`;
						const showLabels = points.length > 1;
						const estimate = vThis.$t("forecast.price.estimate");
						const rows = points.map((s) => ({
							name: vThis.rowLabel(
								showLabels ? s.seriesName : undefined,
								s.data?.estimate ? estimate : undefined
							),
							values: [vThis.fmtPricePerKWh(s.value[1], vThis.currency, true, true)],
						}));
						return tooltipTable(time, rows);
					},
				},
				xAxis: forecastXAxes(
					this.startDate,
					this.endDate,
					this.hourShort,
					this.weekdayShort
				),
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
					...this.priceSeries(
						this.slots,
						priceColor,
						this.$t("main.energyflow.gridImport"),
						this.markPoints
					),
					...this.priceSeries(
						this.feedinSlots,
						colors.export || "",
						this.$t("main.energyflow.pvExport")
					),
				],
			};
		},
	},
	methods: {
		rowLabel(name?: string, hint?: string): string | undefined {
			if (name && hint) return `${name} (${hint})`;
			return name || hint;
		},
		// known prices with gradient, estimates hatched with a thin line
		priceSeries(
			slots: UiForecastSlot[],
			color: string,
			name: string,
			points?: { coord: [number, number]; value: string }[]
		): Record<string, unknown>[] {
			const avg = slots.length ? slots.reduce((a, s) => a + s.value, 0) / slots.length : 0;
			const gradientDown = avg >= 0;
			const known = slots.filter((s) => !s.estimate);
			const estimates = slots.filter((s) => s.estimate);
			const gradient = new echarts.graphic.LinearGradient(
				0,
				gradientDown ? 0 : 1,
				0,
				gradientDown ? 1 : 0,
				[
					{ offset: 0, color: lighterColor(color) || color },
					{ offset: 0.75, color: color + "00" },
					{ offset: 1, color: color + "00" },
				]
			);
			const segment = (part: UiForecastSlot[], estimate: boolean) => {
				const last = part[part.length - 1];
				// known prices close on the first estimate so both parts connect
				const close = (!estimate && estimates[0]) || last;
				const line = estimate ? { width: 1 } : {};
				return {
					name,
					type: "line",
					step: "end",
					cursor: "default",
					...hoverDot(color, line),
					lineStyle: { color, ...lineDefaults, ...line },
					areaStyle: { color: estimate ? hatchPattern(color) : gradient },
					data: [
						...part.map((s) => ({
							value: [clampStart(s.start, this.startDate), s.value],
							estimate,
						})),
						...(last && close
							? [{ value: [last.end, close.value], closing: true }]
							: []),
					],
				};
			};
			const markPoint = points && {
				markPoint: markPointLabel(
					color,
					this.tooltipVisible ? [] : points,
					this.startDate,
					this.endDate
				),
			};
			return [{ ...segment(known, false), ...markPoint }, segment(estimates, true)];
		},
	},
});
</script>
