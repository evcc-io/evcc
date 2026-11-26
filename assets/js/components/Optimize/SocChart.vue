<template>
	<div>
		<div class="legend-row d-flex flex-wrap justify-content-between column-gap-4">
			<LegendList :legends="legends" />
			<LegendList
				v-if="goals.length > 1"
				:legends="laterGoalsLegends"
				v-tooltip="laterGoalsHint"
			/>
		</div>
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
	planGoals?: { time: string; soc: number }[];
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
		// soc is the value at slot end, prefixed with the initial soc at schedule start
		socSeries(): [number, number][] {
			const soc = [
				this.evopt.req.batteries?.[this.entry.index]?.s_initial ?? 0,
				...(this.evopt.res.batteries[this.entry.index]?.state_of_charge || []),
			];
			const capacityWh = this.entry.capacity * 1000;
			return [...this.times, this.endTime].map((t, i) => [
				t,
				capacityWh > 0 ? ((soc[i] ?? 0) / capacityWh) * 100 : 0,
			]);
		},
		// plan goals by time, only the first is fully planned, later ones once they become effective
		goals(): [number, number][] {
			return (this.entry.planGoals || [])
				.map((g): [number, number] => [new Date(g.time).getTime(), g.soc])
				.sort((a, b) => a[0] - b[0]);
		},
		legends(): Legend[] {
			return [{ label: this.entry.title, color: this.entry.color, value: "", type: "area" }];
		},
		laterGoalsLegends(): Legend[] {
			return [
				{
					label: "Plan goals",
					color: this.entry.color,
					value: "",
					type: "line",
				},
			];
		},
		laterGoalsHint(): string {
			return "For now, only the first goal is planned. The plan will adapt once it has passed.";
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
					interval: 25,
					axisLabel: {
						color: colors.muted || "",
						formatter: (v: number) => this.fmtNumber(v, 0),
					},
				}),
				series: [
					{
						type: "line",
						z: 3,
						data: this.socSeries,
						...transientHoverDot(this.entry.color),
						lineStyle: { color: this.entry.color, ...lineDefaults },
						areaStyle: { color: dimColor(this.entry.color) },
						// deadline line from the axis up to the goal SoC, later goals muted
						markLine: {
							data: this.goals.map((coord, i) => [
								{
									coord: [coord[0], 0],
									lineStyle: {
										color: i === 0 ? this.entry.color : colors.muted || "",
									},
								},
								{ coord },
							]),
							symbol: "none",
							lineStyle: { type: "solid", width: 1 },
							label: { show: false },
							silent: true,
						},
						// ring at the goal, value pill at the bottom of the plot
						markPoint: {
							data: this.goals.flatMap(([t, soc], i) => {
								const color = i === 0 ? this.entry.color : colors.muted || "";
								return [
									{
										coord: [t, soc],
										symbol: "circle",
										symbolSize: 8,
										itemStyle: {
											color: colors.background || "",
											borderColor: color,
											borderWidth: 2,
										},
										label: { show: false },
									},
									{
										coord: [t, 0],
										symbolSize: 0,
										value: soc,
										label: { backgroundColor: color },
									},
								];
							}),
							label: {
								show: true,
								position: "top",
								distance: 0,
								fontSize: 10,
								fontWeight: "bold",
								color: colors.background || "",
								backgroundColor: this.entry.color,
								borderRadius: 7,
								padding: [2, 5],
								formatter: (p: { value: number }) => this.fmtPercentage(p.value),
							},
							silent: true,
						},
					},
				],
			};
		},
	},
	methods: {
		tooltipFormatter(
			params: { axisValue: number; dataIndex: number; value?: [number, number] }[]
		): string {
			const arr = Array.isArray(params) ? params : [params];
			const p = arr[0];
			const v = p?.value?.[1];
			if (!p || v == null || Number.isNaN(v)) return "";
			// goal falls into the slot ending at the hovered point
			const prev = this.socSeries[p.dataIndex - 1]?.[0] ?? -Infinity;
			const goal = this.goals.find(([gt]) => gt > prev && gt <= p.axisValue);
			const t = new Date(goal ? goal[0] : p.axisValue);
			const head = `${this.weekdayShort(t)} ${this.fmtHourMinute(t)}`;
			if (!goal) {
				return tooltipTable(head, [{ values: [this.fmtPercentage(v)] }]);
			}
			return tooltipTable(head, [
				{ name: "Charge", values: [this.fmtPercentage(v)] },
				{ name: "Goal", values: [this.fmtPercentage(goal[1])] },
			]);
		},
	},
});
</script>

<style scoped>
/* ends with the plot, grid.right leaves room for the y-axis labels */
.legend-row {
	padding-right: 36px;
}
.soc-chart {
	height: 100px;
	width: 100%;
}
.soc-chart--labels {
	height: 126px;
}
</style>
