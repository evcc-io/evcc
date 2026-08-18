<template>
	<div v-if="chartData.labels.length > 1" class="row">
		<div class="col-12 col-md-6 col-lg-12 col-xxl-6 mb-3">
			<div ref="chartEl" class="round-chart"></div>
		</div>
		<div class="col-12 col-md-6 col-lg-12 col-xxl-6 d-flex align-items-center">
			<LegendList :legends="legends" :device-colors="deviceColors" grid />
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { FONT_FAMILY, topBottomCenterPosition, tooltipStyle, tooltipTable } from "./echarts";
import echartsChart from "@/mixins/echartsChart";
import formatter from "@/mixins/formatter";
import colors, { dimColor } from "@/colors";
import LegendList from "./LegendList.vue";
import { GROUPS, type Session } from "./types.ts";
import type { DeviceColors } from "@/types/evcc";

export default defineComponent({
	name: "SolarGroupedChart",
	components: { LegendList },
	mixins: [formatter, echartsChart],
	props: {
		sessions: { type: Array as PropType<Session[]>, default: () => [] },
		groupBy: {
			type: String as PropType<Exclude<GROUPS, GROUPS.NONE>>,
			default: GROUPS.LOADPOINT,
		},
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
		deviceColors: { type: Object as PropType<DeviceColors>, default: () => ({}) },
	},
	computed: {
		chartData(): { labels: string[]; data: number[]; colors: string[] } {
			const aggregatedData: Record<string, { grid: number; self: number }> = {};

			this.sessions.forEach((session) => {
				const groupKey = session[this.groupBy];
				if (!aggregatedData[groupKey]) {
					aggregatedData[groupKey] = { grid: 0, self: 0 };
				}
				const charged = session.chargedEnergy;
				const self = (charged / 100) * session.solarPercentage;
				const grid = charged - self;
				aggregatedData[groupKey].self += self;
				aggregatedData[groupKey].grid += grid;
			});

			// stable alphabetical order so entries don't jump between periods
			const sortedEntries = Object.entries(aggregatedData).sort((a, b) =>
				a[0].localeCompare(b[0])
			);
			const labels = sortedEntries.map(([label]) => label);
			const data = sortedEntries.map(([, value]) => {
				const total = value.grid + value.self;
				return (value.self / total) * 100;
			});
			const entryColors = labels.map((label) => this.colorMappings[this.groupBy][label]);

			return { labels, data, colors: entryColors };
		},
		legends() {
			const { labels, data, colors: entryColors } = this.chartData;
			return labels.map((label, index) => ({
				label,
				color: entryColors[index],
				value: this.fmtPercentage(data[index] as number, 1),
				id: label || undefined,
			}));
		},
		chartOption(): Record<string, unknown> {
			const { labels, data, colors: entryColors } = this.chartData;
			return {
				animation: false,
				textStyle: { fontFamily: FONT_FAMILY },
				tooltip: {
					trigger: "item",
					...tooltipStyle(colors.text || ""),
					position: topBottomCenterPosition(() => this.chart),
					formatter: (params: { name: string; value: number }) =>
						tooltipTable(params.name, [
							{ values: [this.fmtPercentage(params.value, 1)] },
						]),
				},
				polar: { radius: "95%" },
				angleAxis: { type: "category", data: labels, show: false, startAngle: 90 },
				radiusAxis: {
					min: 0,
					max: 100,
					interval: 25,
					axisLine: { show: false },
					axisTick: { show: false },
					splitLine: { lineStyle: { color: colors.border || "" } },
					axisLabel: {
						fontSize: 10,
						color: colors.muted || "",
						backgroundColor: colors.box || "",
						padding: [1, 3],
					},
				},
				series: [
					{
						type: "bar",
						coordinateSystem: "polar",
						barCategoryGap: "0%",
						data: data.map((value, i) => ({
							value,
							itemStyle: {
								color: dimColor(entryColors[i]),
								borderColor: entryColors[i],
								borderWidth: 3,
								// round outer corners only, sharp apex at the center
								borderRadius: [0, 0, 8, 8],
							},
						})),
					},
				],
			};
		},
	},
});
</script>
