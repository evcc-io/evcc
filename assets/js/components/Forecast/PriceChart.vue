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
	clampStart,
	filterForecastSlots,
	minSlotIndex,
	maxSlotIndex,
} from "./echarts";
import colors from "@/colors";
import formatter from "@/mixins/formatter";
import chartMixin from "./chartMixin";
import type { CURRENCY } from "@/types/evcc";
import type { ForecastSlot } from "./types";

export default defineComponent({
	name: "PriceChart",
	mixins: [formatter, chartMixin],
	props: {
		grid: { type: Array as PropType<ForecastSlot[]>, required: true },
		currency: { type: String as PropType<CURRENCY> },
		zoom: { type: Boolean, default: false },
	},
	data() {
		return {
			hoveredIndex: -1 as number,
		};
	},
	computed: {
		slots(): ForecastSlot[] {
			return filterForecastSlots(this.grid, this.startDate, this.endDate);
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
			const values = this.slots.map((s) => s.value);
			const dataMin = Math.min(...values);
			const dataMax = Math.max(...values);
			const fullMin = Math.min(0, dataMin);
			const fullRange = dataMax - fullMin || 1;
			const rawInterval = fullRange / 5;
			const magnitude = Math.pow(10, Math.floor(Math.log10(rawInterval)));
			const nice = [1, 2, 2.5, 5, 10].find((n) => n * magnitude >= rawInterval) || 10;
			const interval = nice * magnitude;

			if (this.zoom) {
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
			const priceColor = colors.price || "";

			// eslint-disable-next-line @typescript-eslint/no-this-alias
			const vThis = this;
			return {
				animationDuration: 0,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: forecastGrid(),
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
					{
						type: "bar",
						cursor: "default",
						data: this.slots.map((s, i) => ({
							value: [clampStart(s.start, this.startDate), s.value],
							itemStyle:
								this.hoveredIndex >= 0 && i !== this.hoveredIndex
									? { opacity: 0.33 }
									: undefined,
						})),
						barMaxWidth: 4,
						barMinWidth: 4,
						itemStyle: {
							color: priceColor,
							borderRadius: 2,
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
	mounted() {
		this.chart!.on("showTip", (params: unknown) => {
			const p = params as { dataIndex?: number };
			if (p.dataIndex != null) this.hoveredIndex = p.dataIndex;
		});
		this.chart!.on("hideTip", () => {
			this.hoveredIndex = -1;
		});
		this.chart!.getZr().on("mouseout", () => {
			this.tooltipVisible = false;
			this.hoveredIndex = -1;
		});
	},
});
</script>

<style scoped>
.forecast-chart-scroll {
	overflow-x: auto;
	padding-bottom: 4px;
}
</style>
