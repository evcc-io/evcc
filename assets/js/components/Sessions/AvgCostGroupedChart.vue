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

<script>
import { PolarArea } from "vue-chartjs";
import { RadialLinearScale, ArcElement, Legend, Tooltip } from "chart.js";
import { registerChartComponents, commonOptions, tooltipLabelColor } from "./chartConfig";
import formatter from "../../mixins/formatter";
import colors, { dimColor } from "../../colors";
import { TYPES, GROUPS } from "./types";
import LegendList from "./LegendList.vue";

registerChartComponents([RadialLinearScale, ArcElement, Legend, Tooltip]);

export default {
	name: "AvgCostGroupedChart",
	components: { PolarArea, LegendList },
	props: {
		sessions: { type: Array, default: () => [] },
		currency: { type: String, default: "EUR" },
		groupBy: { type: String, default: GROUPS.LOADPOINT },
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
		suggestedMax: { type: Number, default: 0 },
		costType: { type: String, default: TYPES.PRICE },
	},
	mixins: [formatter],
	computed: {
		chartData() {
			console.log(`update ${this.costType} grouped data`);
			const aggregatedData = {};

			this.sessions.forEach((session) => {
				const groupKey = session[this.groupBy];
				if (!aggregatedData[groupKey]) {
					aggregatedData[groupKey] = { energy: 0, cost: 0 };
				}
				const chargedEnergy = session.chargedEnergy;
				if (this.costType === TYPES.CO2) {
					aggregatedData[groupKey].energy += chargedEnergy;
					aggregatedData[groupKey].cost += session.co2PerKWh * chargedEnergy;
				} else if (this.costType === TYPES.PRICE) {
					aggregatedData[groupKey].energy += chargedEnergy;
					aggregatedData[groupKey].cost += session.price;
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
				color: colors.text,
				spacing: 0,
				radius: "100%",
				plugins: {
					...commonOptions.plugins,
					tooltip: {
						...commonOptions.plugins.tooltip,
						intersect: false,
						mode: "index",
						position: "topBottomCenter",
						callbacks: {
							title: () => null,
							label: (tooltipItem) => {
								const { label, raw = 0 } = tooltipItem;
								return (
									label +
									": " +
									(this.costType === TYPES.CO2
										? this.fmtCo2Long(raw)
										: this.fmtPricePerKWh(raw, this.currency))
								);
							},
							labelColor: tooltipLabelColor(true),
						},
					},
				},
				scales: {
					r: {
						suggestedMin: 0,
						suggestedMax: this.suggestedMax,
						beginAtZero: false,
						ticks: {
							color: colors.muted,
							backdropColor: colors.background,
							font: { size: 10 },
							callback: this.formatValue,
							maxTicksLimit: 6,
						},
						angleLines: { display: false },
						grid: { color: colors.border },
					},
				},
			};
		},
	},
	methods: {
		formatValue(value) {
			return this.costType === TYPES.CO2
				? this.fmtCo2Medium(value)
				: this.fmtPricePerKWh(value, this.currency);
		},
	},
};
</script>
