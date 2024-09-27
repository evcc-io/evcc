<template>
	<div v-if="chartData.labels.length > 1">
		<Doughnut :data="chartData" :options="options" />
	</div>
</template>

<script>
import { Doughnut } from "vue-chartjs";
import { Chart, DoughnutController, ArcElement, LinearScale, Legend, Tooltip } from "chart.js";
import formatter from "../../mixins/formatter";
import colors from "../../colors";

Chart.register(DoughnutController, ArcElement, LinearScale, Legend, Tooltip);
Chart.defaults.font.family = window
	.getComputedStyle(document.documentElement)
	.getPropertyValue("--bs-font-sans-serif");

Chart.defaults.font.size = 14;
Chart.defaults.layout.padding = 0;

const { generateLabels } = Chart.overrides.doughnut.plugins.legend.labels;

export default {
	name: "EnergyAggregateChart",
	components: { Doughnut },
	props: {
		sessions: { type: Array, default: () => [] },
		groupBy: { type: String, default: "loadpoint" },
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
	},
	mixins: [formatter],
	computed: {
		chartData() {
			console.log("update energy aggregate data");
			const aggregatedData = {};

			this.sessions.forEach((session) => {
				const groupKey = session[this.groupBy];
				if (!aggregatedData[groupKey]) {
					aggregatedData[groupKey] = 0;
				}
				aggregatedData[groupKey] += session.chargedEnergy;
			});

			// Sort the data by energy in descending order
			const sortedEntries = Object.entries(aggregatedData).sort((a, b) => b[1] - a[1]);

			const labels = sortedEntries.map(([label]) => label);
			const data = sortedEntries.map(([, value]) => value);
			const backgroundColor = labels.map((label) => this.colorMappings[this.groupBy][label]);

			return {
				labels: labels,
				datasets: [{ data, backgroundColor }],
			};
		},
		options() {
			return {
				locale: this.$i18n?.locale,
				responsive: true,
				maintainAspectRatio: false,
				borderRadius: 6,
				color: colors.text,
				plugins: {
					legend: {
						position: "right",
						align: "start",
						labels: {
							usePointStyle: true,
							pointStyle: "circle",
							padding: 20,
							generateLabels: (chart) => {
								const labels = generateLabels(chart);
								const total = chart.data.datasets[0].data.reduce(
									(acc, curr) => acc + curr,
									0
								);
								labels.forEach((label, dataIndex) => {
									const value = chart.data.datasets[0].data[dataIndex];
									const percentage = (100 / total) * value;
									label.text = `${label.text} ${this.fmtPercentage(percentage, 1)}`;
								});
								return labels;
							},
						},
						onClick: () => {},
					},
					tooltip: {
						mode: "index",
						intersect: false,
						callbacks: {
							label: (tooltipItem) => {
								const value = tooltipItem.raw || 0;
								return this.fmtWh(value * 1e3, POWER_UNIT.AUTO);
							},
						},
						backgroundColor: "#000",
					},
				},
				borderWidth: 3,
				borderColor: colors.background,
				cutout: "70%",
				radius: "100%",
				animation: {
					duration: 250,
				},
			};
		},
	},
};
</script>
