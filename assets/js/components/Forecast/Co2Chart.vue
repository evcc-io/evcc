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
import type { ForecastSlot } from "./types";

export default defineComponent({
	name: "Co2Chart",
	mixins: [formatter, chartMixin],
	props: {
		co2: { type: Array as PropType<ForecastSlot[]>, required: true },
	},
	computed: {
		slots(): ForecastSlot[] {
			return filterForecastSlots(this.co2, this.startDate, this.endDate);
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
					value: this.fmtGrams(slots[minIdx]!.value),
					label: { position: "bottom", offset: [0, 2] },
				});
			}
			if (maxIdx !== minIdx && slots[maxIdx]) {
				points.push({
					coord: [clampStart(slots[maxIdx]!.start, this.startDate), slots[maxIdx]!.value],
					value: this.fmtGrams(slots[maxIdx]!.value),
				});
			}
			return points;
		},
		chartOption(): Record<string, unknown> {
			const co2Color = colors.co2 || "";

			// eslint-disable-next-line @typescript-eslint/no-this-alias
			const vThis = this;
			return {
				animationDuration: 0,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: forecastGrid(),
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
				xAxis: forecastXAxes(this.startDate, this.endDate, this.weekdayShort),
				yAxis: forecastYAxis({
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
});
</script>
