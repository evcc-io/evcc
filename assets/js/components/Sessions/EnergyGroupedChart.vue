<template>
	<div class="row" v-if="chartData.labels.length > 1">
		<div class="col-12 col-md-6 mb-3">
			<Doughnut :data="chartData" :options="options" />
		</div>
		<div class="col-12 col-md-6 d-flex align-items-center">
			<LegendList :legends="legends" extra-class="flex-md-column" />
		</div>
	</div>
</template>

<script>
import { Doughnut } from "vue-chartjs";
import { Chart, DoughnutController, ArcElement, LinearScale, Legend, Tooltip } from "chart.js";
import formatter, { POWER_UNIT } from "../../mixins/formatter";
import LegendList from "./LegendList.vue";
import colors from "../../colors";

Chart.register(DoughnutController, ArcElement, LinearScale, Legend, Tooltip);
Chart.defaults.font.family = window
	.getComputedStyle(document.documentElement)
	.getPropertyValue("--bs-font-sans-serif");

Chart.defaults.font.size = 14;
Chart.defaults.layout.padding = 0;

Tooltip.positioners.center = function () {
	const { chart } = this;
	return {
		x: chart.width / 2,
		y: chart.height / 2,
		xAlign: "center",
		yAlign: "center",
	};
};

export default {
	name: "EnergyAggregateChart",
	components: { Doughnut, LegendList },
	props: {
		sessions: { type: Array, default: () => [] },
		groupBy: { type: String, default: "loadpoint" },
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {}, solar: {} }) },
	},
	mixins: [formatter],
	computed: {
		chartData() {
			console.log("update energy aggregate data");
			const aggregatedData = {};

			if (this.groupBy === "solar") {
				const total = this.sessions.reduce((acc, s) => acc + s.chargedEnergy, 0);
				const self = this.sessions.reduce(
					(acc, s) => acc + (s.chargedEnergy / 100) * s.solarPercentage,
					0
				);
				aggregatedData.self = self;
				aggregatedData.grid = total - self;
			} else {
				this.sessions.forEach((session) => {
					const groupKey = session[this.groupBy];
					if (!aggregatedData[groupKey]) {
						aggregatedData[groupKey] = 0;
					}
					aggregatedData[groupKey] += session.chargedEnergy;
				});
			}

			// Sort the data by energy in descending order
			const sortedEntries = Object.entries(aggregatedData); //.sort((a, b) => b[1] - a[1]);

			const labels = sortedEntries.map(([label]) =>
				this.groupBy === "solar" ? this.$t(`sessions.group.${label}`) : label
			);
			const data = sortedEntries.map(([, value]) => value);
			const backgroundColor = sortedEntries.map(
				([label]) => this.colorMappings[this.groupBy][label]
			);

			return {
				labels: labels,
				datasets: [{ data, backgroundColor }],
			};
		},
		legends() {
			const total = this.chartData.datasets[0].data.reduce((acc, curr) => acc + curr, 0);
			return this.chartData.labels.map((label, index) => ({
				label: label,
				color: this.chartData.datasets[0].backgroundColor[index],
				value: this.fmtPercentage(
					(100 / total) * this.chartData.datasets[0].data[index],
					1
				),
			}));
		},
		options() {
			return {
				locale: this.$i18n?.locale,
				responsive: true,
				aspectRatio: 1,
				maintainAspectRatio: false,
				borderRadius: 10,
				color: colors.text,
				plugins: {
					legend: {
						display: false,
					},
					tooltip: {
						mode: "index",
						position: "center",
						intersect: false,
						boxPadding: 5,
						usePointStyle: true,
						borderWidth: 0.00001,
						labelPointStyle: "circle",
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
				radius: "95%",
				animation: {
					duration: 250,
				},
			};
		},
	},
};
</script>
