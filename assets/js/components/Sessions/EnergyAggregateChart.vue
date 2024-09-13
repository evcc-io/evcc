<template>
	<div style="position: relative; height: 200px">
		<Doughnut :data="chartData" :options="options" />
	</div>
</template>

<script>
import { Doughnut } from "vue-chartjs";
import { Chart, DoughnutController, ArcElement, LinearScale, Legend, Tooltip } from "chart.js";
import formatter from "../../mixins/formatter";

Chart.register(DoughnutController, ArcElement, LinearScale, Legend, Tooltip);
Chart.defaults.font.family = window
	.getComputedStyle(document.documentElement)
	.getPropertyValue("--bs-font-sans-serif");

Chart.defaults.font.size = 14;
Chart.defaults.layout.padding = 0;
Chart.defaults.animation = false;

export default {
	name: "EnergyAggregateChart",
	components: { Doughnut },
	props: {
		sessions: { type: Array, default: () => [] },
		groupBy: { type: String, default: "loadpoint" },
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
	},
	data() {
		return {
			borderColor: null,
		};
	},
	mixins: [formatter],
	mounted() {
		window
			.matchMedia("(prefers-color-scheme: dark)")
			.addEventListener("change", this.updateColors);
		this.updateColors();
	},
	beforeUnmount() {
		window
			.matchMedia("(prefers-color-scheme: dark)")
			.removeEventListener("change", this.updateColors);
	},
	methods: {
		updateColors() {
			const style = window.getComputedStyle(document.documentElement);
			this.borderColor = style.getPropertyValue("--evcc-background");
		},
	},
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
				animations: true,
				locale: this.$i18n?.locale,
				responsive: true,
				maintainAspectRatio: false,
				borderRadius: 6,
				plugins: {
					legend: { display: false },
					tooltip: {
						mode: "index",
						intersect: false,
						callbacks: {
							label: (tooltipItem) => {
								const datasetLabel = tooltipItem.dataset.label || "";
								const value = tooltipItem.raw;
								return value
									? datasetLabel + ": " + this.fmtKWh(value * 1e3)
									: null;
							},
						},
						backgroundColor: "#000",
					},
				},
				borderWidth: 3,
				borderColor: this.borderColor,
				cutout: "70%",
				radius: "100%",
			};
		},
	},
};
</script>
