<template>
	<div
		ref="chartEl"
		class="pattern"
		:style="{ height: `${height}px` }"
		data-testid="pattern-chart"
	></div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { FONT_FAMILY, forecastGrid, tooltipStyle, tooltipTable, xAxisLabelStyle } from "./echarts";
import echartsChart from "@/mixins/echartsChart";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import colors, { setAlpha } from "@/colors";
import { SLOT_MS, slotWatts } from "./slots";
import type { HistorySeries } from "./GroupChart.vue";
import { PERIODS } from "../Sessions/types";
import { labelStep, DAY_STEPS, MONTH_STEPS } from "@/utils/labelStep";
import { is12hFormat } from "@/units";

// energy of one entity as a pattern: a row of 15 minute cells for a day, hour by day for a month, day calendar for longer ranges
export default defineComponent({
	name: "PatternChart",
	mixins: [formatter, echartsChart],
	props: {
		series: { type: Object as PropType<HistorySeries>, required: true },
		color: { type: String, required: true },
		from: { type: Date, required: true },
		to: { type: Date, required: true },
		period: { type: String as PropType<PERIODS>, required: true },
		// same as the bar chart so toggling does not jump
		height: { type: Number, default: 260 },
	},
	data() {
		return { chartWidth: 0 };
	},
	computed: {
		max(): number {
			return Math.max(0, ...this.series.data.map((slot) => slot.energy));
		},
		chartOption(): Record<string, unknown> {
			return {
				animation: false,
				textStyle: { fontFamily: FONT_FAMILY },
				...this.periodOption,
			};
		},
		visualMap(): Record<string, unknown> {
			return {
				show: false,
				min: 0,
				max: this.max || 1,
				inRange: { color: [setAlpha(this.color, "18"), this.color] },
				// the legend gradient would otherwise blend the default gray into the light end
				outOfRange: { color: "transparent" },
			};
		},
		periodOption(): Record<string, unknown> {
			switch (this.period) {
				case PERIODS.DAY:
					return this.slotRowOption;
				case PERIODS.MONTH:
					return this.hourDayOption;
				default:
					return this.calendarOption;
			}
		},
		// day: one row of 15 minute cells over the bar chart's frame, color by energy,
		// with the color scale above
		slotRowOption(): Record<string, unknown> {
			const slots = 96;
			const data = this.series.data.map((slot) => [
				this.slotIndex(slot.start),
				0,
				slot.energy,
			]);
			const step = labelStep(24, this.chartWidth, DAY_STEPS, is12hFormat() ? 56 : 40);
			const maxLabel = this.fmtW(slotWatts(this.max), POWER_UNIT.AUTO);
			return {
				tooltip: {
					...tooltipStyle(colors.text || ""),
					formatter: (params: { value: [number, number, number] }) => {
						const start = new Date(this.from.getTime() + params.value[0] * SLOT_MS);
						// 15 minute slot as average power, like the bar chart
						return tooltipTable(this.fmtTimeSlot(start, SLOT_MS), [
							{ values: [this.fmtW(slotWatts(params.value[2]), POWER_UNIT.AUTO)] },
						]);
					},
				},
				visualMap: this.legend(maxLabel, 36),
				// same frame as the bar chart so toggling does not jump
				// the bar chart reserves a left axis for soc or temperature, same frame here
				grid: {
					...forecastGrid(),
					left: this.series.data.some((slot) => slot.socTemp != null) ? 36 : 0,
					right: 36,
				},
				xAxis: {
					type: "category",
					data: Array.from({ length: slots }, (_, i) => i),
					axisLine: { show: false },
					axisTick: { show: false },
					axisLabel: {
						...xAxisLabelStyle(),
						interval: 0,
						// full hours in even steps, 00:00 skipped like the bar chart
						formatter: (_: string, i: number) =>
							i > 0 && i % (4 * step) === 0
								? this.hourShort(new Date(this.from.getTime() + i * SLOT_MS))
								: "",
					},
				},
				yAxis: { type: "category", data: [""], show: false },
				series: [
					{
						type: "heatmap",
						data,
						itemStyle: {
							borderWidth: 1,
							borderColor: colors.box || "",
							borderRadius: 2,
						},
						emphasis: { disabled: true },
					},
				],
			};
		},
		dayStep(): number {
			return labelStep(31, this.chartWidth, MONTH_STEPS);
		},
		// month: hour of day over day of month
		hourDayOption(): Record<string, unknown> {
			const days = Math.round((this.to.getTime() - this.from.getTime()) / 864e5);
			const data = this.series.data.map((slot) => {
				const d = new Date(slot.start);
				return [d.getDate() - 1, d.getHours(), slot.energy];
			});
			return {
				tooltip: {
					...tooltipStyle(colors.text || ""),
					formatter: (params: { value: [number, number, number] }) => {
						const d = new Date(
							this.from.getFullYear(),
							this.from.getMonth(),
							params.value[0] + 1
						);
						return tooltipTable(
							`${this.fmtDayMonth(d)} · ${String(params.value[1]).padStart(2, "0")}:00`,
							[{ values: [this.fmtKWh(params.value[2])] }]
						);
					},
				},
				visualMap: this.legend(this.fmtKWh(this.max), 36),
				// same frame and label styles as the bar chart
				grid: { ...forecastGrid(), left: 0, right: 36 },
				xAxis: {
					type: "category",
					data: Array.from({ length: days }, (_, i) => i + 1),
					axisLine: { show: false },
					axisTick: { show: false },
					// even steps like the bar chart instead of dropping overlapping labels
					axisLabel: {
						...xAxisLabelStyle(),
						hideOverlap: false,
						interval: this.dayStep - 1,
					},
				},
				yAxis: {
					type: "category",
					position: "right",
					data: Array.from({ length: 24 }, (_, i) => i),
					inverse: true,
					axisLine: { show: false },
					axisTick: { show: false },
					axisLabel: {
						fontSize: 10,
						opacity: 0.75,
						color: colors.muted || "",
						interval: 5,
					},
				},
				series: [
					{
						type: "heatmap",
						data,
						itemStyle: {
							borderWidth: 1,
							borderColor: colors.box || "",
							borderRadius: 2,
						},
						emphasis: { disabled: true },
					},
				],
			};
		},
		// year: github style calendar of days
		calendarOption(): Record<string, unknown> {
			const last = new Date(this.to.getTime() - 1);
			const data = this.series.data.map((slot) => [
				this.dayKey(new Date(slot.start)),
				slot.energy,
			]);
			return {
				tooltip: {
					...tooltipStyle(colors.text || ""),
					formatter: (params: { value: [string, number] }) =>
						tooltipTable(this.fmtDayMonthYear(new Date(params.value[0])), [
							{ values: [this.fmtKWh(params.value[1])] },
						]),
				},
				visualMap: this.legend(this.fmtKWh(this.max), 4),
				calendar: {
					left: 28,
					right: 4,
					// below the legend
					top: 44,
					bottom: 4,
					range: [this.dayKey(this.from), this.dayKey(last)],
					cellSize: ["auto", "auto"],
					splitLine: { show: false },
					itemStyle: {
						color: "transparent",
						borderWidth: 2,
						borderColor: colors.box || "",
					},
					dayLabel: {
						firstDay: 1,
						nameMap: this.dayNames,
						fontSize: 10,
						color: colors.muted || "",
						margin: 6,
					},
					monthLabel: { fontSize: 10, color: colors.muted || "", margin: 6 },
					yearLabel: { show: false },
				},
				series: [
					{
						type: "heatmap",
						coordinateSystem: "calendar",
						data,
						itemStyle: { borderRadius: 3 },
						emphasis: { disabled: true },
					},
				],
			};
		},
		dayNames(): string[] {
			// echarts expects sunday first
			return [0, 1, 2, 3, 4, 5, 6].map((i) => this.fmtWeekdayByIndex(i, "narrow"));
		},
	},
	methods: {
		// the color scale top right in the frame's header space, its max label ending
		// at the plot edge
		legend(maxLabel: string, margin: number): Record<string, unknown> {
			return {
				...this.visualMap,
				dimension: 2,
				show: true,
				hoverLink: false,
				orient: "horizontal",
				right: margin,
				// text on the same line as the axis name
				top: 8,
				padding: 0,
				itemWidth: 8,
				itemHeight: 80,
				text: [maxLabel, "0"],
				textStyle: { fontSize: 10, color: setAlpha(colors.muted, "bf") || "" },
			};
		},
		slotIndex(start: string): number {
			const d = new Date(start);
			return d.getHours() * 4 + Math.floor(d.getMinutes() / 15);
		},
		onChartInit() {
			this.chartWidth = this.chart?.getWidth() ?? 0;
		},
		resize() {
			this.chart?.resize();
			this.chartWidth = this.chart?.getWidth() ?? 0;
		},
		// the period options have different coordinate systems, replace instead of merge
		applyChartOption() {
			this.chart?.resize();
			this.chartWidth = this.chart?.getWidth() ?? 0;
			this.chart?.setOption(this.chartOption, { notMerge: true });
		},
		dayKey(d: Date): string {
			const p = (n: number) => String(n).padStart(2, "0");
			return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
		},
	},
});
</script>

<style scoped>
.pattern {
	width: 100%;
}
</style>
