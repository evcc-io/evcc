<template>
	<div>
		<LegendList
			:legends="[{ label: entry.title, color: entry.color, value: '', type: 'area' }]"
		/>
		<div ref="chartEl" class="soc-chart" :class="{ 'soc-chart--labels': showXAxis }"></div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import {
	FONT_FAMILY,
	forecastXAxes,
	forecastYAxis,
	tooltipStyle,
	tooltipTable,
	hoverDot,
	lineDefaults,
} from "../Forecast/echarts";
import colors, { dimColor } from "@/colors";
import formatter from "@/mixins/formatter";
import echartsChart from "@/mixins/echartsChart";
import { is12hFormat } from "@/units";
import type { EvoptData } from "./TimeSeriesDataTable.vue";
import LegendList from "../Sessions/LegendList.vue";
import { slotTimes } from "./chart";

export interface SocChartEntry {
	index: number; // index into evopt.res.batteries
	title: string;
	capacity: number; // kWh
	color: string;
}

export default defineComponent({
	name: "SocChart",
	components: { LegendList },
	mixins: [formatter, echartsChart],
	props: {
		evopt: {
			type: Object as PropType<EvoptData>,
			required: true,
		},
		entry: {
			type: Object as PropType<SocChartEntry>,
			required: true,
		},
		timestamp: {
			type: String,
			default: "",
		},
		showXAxis: { type: Boolean, default: false },
	},
	computed: {
		times(): number[] {
			return slotTimes(this.timestamp, this.evopt.req.time_series.dt);
		},
		endTime(): number {
			const dt = this.evopt.req.time_series.dt;
			const last = this.times[this.times.length - 1];
			return last === undefined ? 0 : last + (dt[dt.length - 1] || 0) * 1000;
		},
		socSeries(): [number, number][] {
			const soc = this.evopt.res.batteries[this.entry.index]?.state_of_charge || [];
			const capacityWh = this.entry.capacity * 1000;
			return this.times.map((t, i) => {
				const wh = soc[i] ?? 0;
				return [t, capacityWh > 0 ? (wh / capacityWh) * 100 : 0] as [number, number];
			});
		},
		xAxes(): Record<string, unknown>[] {
			const stepHours = is12hFormat() ? 6 : 4;
			const [hourAxis, dayAxis] = forecastXAxes(
				this.times[0] ?? 0,
				this.endTime,
				this.hourShort,
				this.weekdayShort,
				stepHours
			);
			if (!this.showXAxis) {
				(hourAxis as Record<string, unknown>)["axisLabel"] = { show: false };
			}
			return [hourAxis, dayAxis];
		},
		chartOption(): Record<string, unknown> {
			return {
				animation: false,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: {
					top: 10,
					right: 36,
					bottom: this.showXAxis ? 34 : 8,
					left: 0,
					borderWidth: 0,
				},
				tooltip: {
					trigger: "axis",
					axisPointer: {
						type: "line",
						lineStyle: { color: colors.muted || "", opacity: 0.4 },
					},
					...tooltipStyle(colors.text || ""),
					formatter: this.tooltipFormatter,
				},
				xAxis: this.xAxes,
				yAxis: forecastYAxis({
					min: 0,
					max: 100,
					position: "right",
					interval: 50,
					axisLabel: {
						color: colors.muted || "",
						formatter: (v: number) => this.fmtNumber(v, 0),
					},
				}),
				series: [
					{
						type: "line",
						step: "start",
						z: 3,
						data: this.socSeries,
						...hoverDot(this.entry.color),
						lineStyle: { color: this.entry.color, ...lineDefaults },
						areaStyle: { color: dimColor(this.entry.color) },
					},
				],
			};
		},
	},
	methods: {
		tooltipFormatter(params: { axisValue: number; value?: [number, number] }[]): string {
			const arr = Array.isArray(params) ? params : [params];
			const v = arr[0]?.value?.[1];
			if (v == null || Number.isNaN(v)) return "";
			const t = new Date(arr[0]!.axisValue);
			const head = `${this.weekdayShort(t)} ${this.fmtHourMinute(t)}`;
			return tooltipTable(head, [{ values: [this.fmtPercentage(v)] }]);
		},
	},
});
</script>

<style scoped>
.soc-chart {
	height: 54px;
	width: 100%;
}
.soc-chart--labels {
	height: 80px;
}
</style>
