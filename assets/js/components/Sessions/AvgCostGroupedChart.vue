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
import {
	FONT_FAMILY,
	lineDefaults,
	topBottomCenterPosition,
	tooltipStyle,
	tooltipTable,
} from "./echarts";
import echartsChart from "@/mixins/echartsChart";
import formatter from "@/mixins/formatter";
import colors, { dimColor } from "@/colors";
import LegendList from "./LegendList.vue";
import type { CURRENCY, DeviceColors } from "@/types/evcc";
import { TYPES, GROUPS, type Session } from "./types.ts";

export default defineComponent({
	name: "AvgCostGroupedChart",
	components: { LegendList },
	mixins: [formatter, echartsChart],
	props: {
		sessions: { type: Array as PropType<Session[]>, default: () => [] },
		currency: { type: String as PropType<CURRENCY>, default: "EUR" },
		groupBy: {
			type: String as PropType<Exclude<GROUPS, GROUPS.NONE>>,
			default: GROUPS.LOADPOINT,
		},
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
		deviceColors: { type: Object as PropType<DeviceColors>, default: () => ({}) },
		costType: { type: String as PropType<TYPES>, default: TYPES.PRICE },
	},
	computed: {
		chartData(): { labels: string[]; data: number[]; colors: string[] } {
			const aggregatedData: Record<string, { energy: number; cost: number }> = {};

			this.sessions.forEach((session) => {
				const groupKey = session[this.groupBy];
				if (!aggregatedData[groupKey]) {
					aggregatedData[groupKey] = { energy: 0, cost: 0 };
				}
				const chargedEnergy = session.chargedEnergy;
				if (this.costType === TYPES.CO2) {
					aggregatedData[groupKey].energy += chargedEnergy;
					aggregatedData[groupKey].cost += (session.co2PerKWh || 0) * chargedEnergy;
				} else if (this.costType === TYPES.PRICE) {
					aggregatedData[groupKey].energy += chargedEnergy;
					aggregatedData[groupKey].cost += session.price || 0;
				}
			});

			// stable alphabetical order so entries don't jump between periods
			const sortedEntries = Object.entries(aggregatedData).sort((a, b) =>
				a[0].localeCompare(b[0])
			);
			const labels = sortedEntries.map(([label]) => label);
			const data = sortedEntries.map(([, value]) => value.cost / value.energy);
			const entryColors = labels.map((label) => this.colorMappings[this.groupBy][label]);

			return { labels, data, colors: entryColors };
		},
		legends() {
			const { labels, data, colors: entryColors } = this.chartData;
			return labels.map((label, index) => ({
				label,
				color: entryColors[index],
				value: this.formatValue(data[index] as number),
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
							{
								values: [
									this.costType === TYPES.CO2
										? this.fmtCo2Long(params.value)
										: this.fmtPricePerKWh(params.value, this.currency),
								],
							},
						]),
				},
				polar: { radius: "95%" },
				angleAxis: { type: "category", data: labels, show: false, startAngle: 90 },
				radiusAxis: {
					min: 0,
					splitNumber: 4,
					axisLine: { show: false },
					axisTick: { show: false },
					splitLine: { lineStyle: { color: colors.border || "" } },
					axisLabel: {
						fontSize: 10,
						color: colors.muted || "",
						backgroundColor: colors.box || "",
						padding: [1, 3],
						formatter: this.formatValue,
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
								borderWidth: lineDefaults.width,
								// round outer corners only, sharp apex at the center
								borderRadius: [0, 0, 8, 8],
							},
						})),
					},
				],
			};
		},
	},
	methods: {
		formatValue(value: number) {
			return this.costType === TYPES.CO2
				? this.fmtCo2Medium(value)
				: this.fmtPricePerKWh(value, this.currency);
		},
	},
});
</script>
