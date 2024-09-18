<template>
	<div v-if="chartData.labels.length > 1">
		<PolarArea :data="chartData" :options="options" />
	</div>
</template>

<script>
import { PolarArea } from "vue-chartjs";
import { Chart, RadialLinearScale, ArcElement, Legend, Tooltip } from "chart.js";
import formatter from "../../mixins/formatter";
import colors, { dimColor } from "../../colors";

Chart.register(RadialLinearScale, ArcElement, Legend, Tooltip);
Chart.defaults.font.family = window
	.getComputedStyle(document.documentElement)
	.getPropertyValue("--bs-font-sans-serif");

Chart.defaults.font.size = 14;
Chart.defaults.layout.padding = 0;

const { generateLabels } = Chart.overrides.pie.plugins.legend.labels;

export default {
	name: "SolarChart",
	components: { PolarArea },
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
					aggregatedData[groupKey] = { grid: 0, self: 0 };
				}
				const charged = session.chargedEnergy;
				const self = (charged / 100) * session.solarPercentage;
				const grid = charged - self;
				aggregatedData[groupKey].self += self;
				aggregatedData[groupKey].grid += grid;
			});

			// Sort the data by total energy in descending order
			const sortedEntries = Object.entries(aggregatedData).sort(
				(a, b) => b[1].grid + b[1].self - (a[1].grid + a[1].self)
			);
			const labels = sortedEntries.map(([label]) => label);
			const data = sortedEntries.map(([, value]) => {
				const total = value.grid + value.self;
				const selfPercentage = (value.self / total) * 100;
				return selfPercentage;
			});

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
		options() {
			return {
				locale: this.$i18n?.locale,
				responsive: true,
				maintainAspectRatio: false,
				borderRadius: 8,
				borderWidth: 3,
				color: colors.text,
				spacing: 0,
				plugins: {
					legend: {
						position: "right",
						align: "start",
						boxWidth: 300,
						labels: {
							usePointStyle: true,
							pointStyle: "circle",
							padding: 20,
							generateLabels: (chart) => {
								const labels = generateLabels(chart);
								labels.forEach((label, dataIndex) => {
									console.log(label);
									const value = chart.data.datasets[0].data[dataIndex];
									label.text = `${label.text} ${this.fmtPercentage(value, 1)}`;
									label.fillStyle = label.strokeStyle;
									label.strokeStyle = colors.background;
								});
								return labels;
							},
						},
						onClick: () => {},
					},
				},
				scales: {
					r: {
						min: 0,
						max: 100,
						ticks: {
							stepSize: 25,
							color: colors.muted,
							backdropColor: colors.background,
						},
						grid: { color: colors.border },
					},
				},
				radius: "100%",
			};
		},
	},
};
</script>
