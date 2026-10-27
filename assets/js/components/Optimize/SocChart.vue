<template>
	<div>
		<div ref="chartEl" class="soc-chart my-3"></div>
		<LegendList :legends="legends" />
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
	type TooltipRow,
	lineDefaults,
} from "../Forecast/echarts";
import colors, { dimColor } from "@/colors";
import formatter from "@/mixins/formatter";
import echartsChart from "@/mixins/echartsChart";
import { is12hFormat } from "@/units";
import type { EvoptData } from "./TimeSeriesDataTable.vue";
import LegendList from "../Sessions/LegendList.vue";
import type { Legend } from "../Sessions/types";
import { slotTimes, transientHoverDot } from "./chart";

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
		entries: {
			type: Array as PropType<SocChartEntry[]>,
			required: true,
		},
		timestamp: {
			type: String,
			default: "",
		},
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
		chartSeries(): Record<string, unknown>[] {
			return this.entries.map((entry) => ({
				name: entry.title,
				type: "line",
				z: 3,
				data: this.socSeries(entry),
				...transientHoverDot(entry.color),
				lineStyle: { color: entry.color, ...lineDefaults },
				areaStyle: { color: dimColor(entry.color) },
			}));
		},
		legends(): Legend[] {
			return this.entries.map((entry) => ({
				label: entry.title,
				color: entry.color,
				value: "",
				type: "line",
			}));
		},
		chartOption(): Record<string, unknown> {
			return {
				animation: false,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: { top: 10, right: 36, bottom: 34, left: 0, borderWidth: 0 },
				tooltip: {
					trigger: "axis",
					axisPointer: {
						type: "line",
						lineStyle: { color: colors.muted || "", opacity: 0.4 },
					},
					...tooltipStyle(colors.text || ""),
					formatter: this.tooltipFormatter,
				},
				xAxis: forecastXAxes(
					this.times[0] ?? 0,
					this.endTime,
					this.hourShort,
					this.weekdayShort,
					is12hFormat() ? 6 : 4
				),
				yAxis: forecastYAxis({
					min: 0,
					max: 100,
					position: "right",
					interval: 25,
					axisLabel: {
						color: colors.muted || "",
						formatter: (v: number) => this.fmtNumber(v, 0),
					},
				}),
				series: this.chartSeries,
			};
		},
	},
	methods: {
		// soc is the value at slot end, prefixed with the initial soc at schedule start
		socSeries(entry: SocChartEntry): [number, number][] {
			const soc = [
				this.evopt.req.batteries?.[entry.index]?.s_initial ?? 0,
				...(this.evopt.res.batteries[entry.index]?.state_of_charge || []),
			];
			const capacityWh = entry.capacity * 1000;
			return [...this.times, this.endTime].map((t, i) => [
				t,
				capacityWh > 0 ? ((soc[i] ?? 0) / capacityWh) * 100 : 0,
			]);
		},
		tooltipFormatter(
			params: { axisValue: number; seriesName?: string; value?: [number, number] }[]
		): string {
			const arr = Array.isArray(params) ? params : [params];
			if (!arr.length) return "";
			const t = new Date(arr[0]!.axisValue);
			const head = `${this.weekdayShort(t)} ${this.fmtHourMinute(t)}`;
			const rows: TooltipRow[] = arr
				.filter((p) => p.value?.[1] != null && !Number.isNaN(p.value[1]))
				.map((p) => ({ name: p.seriesName, values: [this.fmtPercentage(p.value![1])] }));
			return tooltipTable(head, rows);
		},
	},
});
</script>

<style scoped>
.soc-chart {
	height: 200px;
	width: 100%;
}
</style>
