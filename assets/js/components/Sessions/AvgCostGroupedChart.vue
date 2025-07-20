<template>
	<div v-if="chartData.labels.length > 1" class="row">
		<div class="col-12 col-md-6 mb-3">
			<PolarArea :data="chartData" :options="options" />
		</div>
		<div class="col-12 col-md-6 d-flex align-items-center py-0 py-md-3">
			<LegendList :legends="legends" grid />
		</div>
	</div>
</template>

<script lang="ts">
import { PolarArea } from "vue-chartjs";
import { RadialLinearScale, ArcElement, Legend, Tooltip, type TooltipItem } from "chart.js";
import { registerChartComponents, commonOptions, tooltipLabelColor } from "./chartConfig.ts";
import formatter from "@/mixins/formatter";
import colors, { dimColor } from "@/colors";
import LegendList from "./LegendList.vue";
import { defineComponent, type PropType } from "vue";
import type { CURRENCY } from "@/types/evcc";
import { TYPES, GROUPS, type Session } from "./types.ts";

registerChartComponents([RadialLinearScale, ArcElement, Legend, Tooltip]);

export default defineComponent({
	name: "AvgCostGroupedChart",
	components: { PolarArea, LegendList },
	mixins: [formatter],
	props: {
		sessions: { type: Array as PropType<Session[]>, default: () => [] },
		currency: { type: String as PropType<CURRENCY>, default: "EUR" },
		groupBy: {
			type: String as PropType<Exclude<GROUPS, GROUPS.NONE>>,
			default: GROUPS.LOADPOINT,
		},
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
		suggestedMax: { type: Number, default: 0 },
		costType: { type: String as PropType<TYPES>, default: TYPES.PRICE },
	},
	computed: {
		chartData() {
			console.log(`update ${this.costType} grouped data`);
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

			const sortedEntries = Object.entries(aggregatedData).sort(
				(a, b) => b[1].cost - a[1].cost
			);
			const labels = sortedEntries.map(([label]) => label);
			const data = sortedEntries.map(([, value]) => value.cost / value.energy);

			const borderColors = labels.map((label) => this.colorMappings[this.groupBy][label]);
			const backgroundColors = borderColors.map((color) => dimColor(color));
			return {
				labels: labels,
				datasets: [
					{
						data: data,
						borderColor: borderColors,
						backgroundColor: backgroundColors,
					},
				],
			};
		},
		legends() {
			return this.chartData.labels.map((label, index) => ({
				label: label,
				color: this.chartData.datasets[0].borderColor[index],
				value: this.formatValue(this.chartData.datasets[0].data[index]),
			}));
		},
		options() {
			return {
				...commonOptions,
				locale: this.$i18n?.locale,
				aspectRatio: 1,
				borderRadius: 8,
				borderWidth: 3,
				color: colors.text || "",
				spacing: 0,
				radius: "100%",
				plugins: {
					...commonOptions.plugins,
					tooltip: {
						...commonOptions.plugins.tooltip,
						axis: "r",
						position: "topBottomCenter",
						callbacks: {
							title: () => null,
							label: (tooltipItem: TooltipItem<"polarArea">) => {
								const { label, dataset, dataIndex } = tooltipItem;
								const d = dataset.data[dataIndex];

								return (
									label +
									": " +
									(this.costType === TYPES.CO2
										? this.fmtCo2Long(d)
										: this.fmtPricePerKWh(d, this.currency))
								);
							},
							labelColor: tooltipLabelColor(true),
						},
					} as any,
				},
				scales: {
					r: {
						suggestedMin: 0,
						suggestedMax: this.suggestedMax,
						beginAtZero: false,
						ticks: {
							color: colors.muted || "",
							backdropColor: colors.background || "",
							font: { size: 10 },
							callback: this.formatValue,
							maxTicksLimit: 6,
						},
						angleLines: { display: false },
						grid: { color: colors.border || "" },
					} as any,
				},
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
