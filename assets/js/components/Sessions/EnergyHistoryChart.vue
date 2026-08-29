<template>
	<div>
		<div ref="chartEl" class="chart my-3"></div>
		<LegendList :legends="legends" :device-colors="deviceColors" />
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import {
	FONT_FAMILY,
	forecastGrid,
	forecastYAxis,
	roundedStackData,
	tooltipStyle,
	tooltipTable,
	xAxisLabelStyle,
	type TooltipRow,
} from "./echarts";
import historyChart from "./historyChart";
import LegendList from "./LegendList.vue";
import { POWER_UNIT } from "@/mixins/formatter";
import colors from "@/colors";
import { GROUPS } from "./types";
import type { DeviceColors } from "@/types/evcc";

interface Dataset {
	label: string;
	color: string;
	data: number[];
}

export default defineComponent({
	name: "EnergyHistoryChart",
	components: { LegendList },
	mixins: [historyChart],
	props: {
		groupBy: { type: String as PropType<GROUPS>, default: GROUPS.NONE },
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
		deviceColors: { type: Object as PropType<DeviceColors>, default: () => ({}) },
	},
	computed: {
		chartData(): { labels: string[]; datasets: Dataset[] } {
			const result: Record<number, Record<string, number>> = {};
			const groups: Set<string> = new Set();

			const range = this.bucketRange;
			if (range) {
				for (let i = range[0]; i <= range[1]; i++) {
					result[i] = {};
				}

				this.sessions.forEach((session) => {
					const index = this.bucketIndex(new Date(session.created));

					if (this.groupBy === GROUPS.NONE) {
						groups.add("grid");
						groups.add("self");
						const charged = session.chargedEnergy;
						const self = (charged / 100) * session.solarPercentage;
						const grid = charged - self;
						const item = result[index]!;
						item["self"] = (item["self"] || 0) + self;
						item["grid"] = (item["grid"] || 0) + grid;
					} else {
						const groupKey = session[this.groupBy];
						groups.add(groupKey);
						result[index]![groupKey] =
							(result[index]![groupKey] || 0) + session.chargedEnergy;
					}
				});
			}

			// stable alphabetical order so entries don't jump between periods
			const sortedGroups = Array.from(groups).sort((a, b) => a.localeCompare(b));

			const datasets = sortedGroups.map((group) => {
				const colorGroup = this.groupBy === GROUPS.NONE ? "solar" : this.groupBy;
				const color = this.colorMappings[colorGroup][group];
				const label =
					this.groupBy === GROUPS.NONE ? this.$t(`sessions.group.${group}`) : group;
				return {
					color,
					label,
					data: Object.values(result).map((day) => day[group] || 0),
				};
			});

			return {
				labels: Object.keys(result),
				datasets,
			};
		},
		legends() {
			const pickable = this.groupBy !== GROUPS.NONE;
			return this.chartData.datasets.map((dataset) => ({
				label: dataset.label,
				color: dataset.color,
				value: this.fmtWh(
					dataset.data.reduce((acc, curr) => acc + curr, 0) * 1e3,
					POWER_UNIT.AUTO
				),
				id: pickable ? dataset.label || undefined : undefined,
			}));
		},
		chartOption(): Record<string, unknown> {
			const { labels, datasets } = this.chartData;
			const head = this.tooltipHead;
			const stackData = roundedStackData(datasets, labels.length);
			return {
				animation: false,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: { ...forecastGrid(), left: 0, right: 36 },
				tooltip: {
					trigger: "axis",
					axisPointer: { type: "shadow", shadowStyle: { color: "transparent" } },
					...tooltipStyle(colors.text || ""),
					// read values from chartData, rendered series hide tiny top slivers
					formatter: (params: { dataIndex: number }[]) => {
						const idx = params?.[0]?.dataIndex;
						if (idx == null) return "";
						const rows: TooltipRow[] = [];
						const values = datasets.map((d) => d.data[idx] || 0).filter((v) => v > 0);
						if (!values.length) return "";
						const unit = this.getPowerUnit(Math.max(...values) * 1000);
						// top of stack first
						[...datasets].reverse().forEach((dataset) => {
							const value = dataset.data[idx] || 0;
							if (!value) return;
							rows.push({
								name: datasets.length > 1 ? dataset.label : undefined,
								values: [this.fmtWh(value * 1e3, unit)],
							});
						});
						return tooltipTable(head(labels[idx] ?? ""), rows);
					},
				},
				xAxis: {
					type: "category",
					data: labels,
					axisLine: { show: false },
					axisTick: { show: false },
					splitLine: { show: false },
					axisLabel: {
						...xAxisLabelStyle(),
						formatter: this.xAxisLabel,
					},
				},
				yAxis: forecastYAxis({
					position: "right",
					min: 0,
					splitLine: {
						showMinLine: true,
						showMaxLine: true,
						lineStyle: { color: colors.border || "" },
					},
					name: "kWh",
					nameLocation: "end",
					nameGap: 18,
					nameTextStyle: {
						color: colors.muted || "",
						fontFamily: FONT_FAMILY,
						fontSize: 10,
						opacity: 0.75,
						align: "left",
						// align with the value labels' left edge (8px default label margin)
						padding: [0, 0, 0, 8],
					},
					axisLabel: {
						color: colors.muted || "",
						hideOverlap: true,
						formatter: (v: number) => this.fmtWh(v * 1e3, POWER_UNIT.KW, false, 0),
					},
				}),
				series: datasets.map((dataset, i) => ({
					name: dataset.label,
					type: "bar",
					stack: "energy",
					barMaxWidth: 40,
					itemStyle: { color: dataset.color },
					data: stackData[i],
				})),
			};
		},
	},
});
</script>

<style scoped>
.chart {
	width: 100%;
	height: 300px;
}
</style>
