<template>
	<div style="position: relative; height: 300px">
		<PolarArea :data="chartData" :options="options" />
	</div>
</template>

<script>
import { PolarArea } from "vue-chartjs";
import { Chart, PolarAreaController, CategoryScale, LinearScale, Legend, Tooltip } from "chart.js";
import formatter from "../../mixins/formatter";

Chart.register(PolarAreaController, CategoryScale, LinearScale, Legend, Tooltip);
Chart.defaults.font.family = window
	.getComputedStyle(document.documentElement)
	.getPropertyValue("--bs-font-sans-serif");

Chart.defaults.font.size = 14;
Chart.defaults.layout.padding = 0;
Chart.defaults.animation = false;

const loadpointColors = [
	"#0077B6FF",
	"#00B4D8FF",
	"#90E0EFFF",
	"#006769FF",
	"#40A578FF",
	"#9DDE8BFF",
	"#F8961EFF",
	"#F9C74FFF",
	"#E6FF94FF",
];

export default {
	name: "SolarPolarChart",
	components: { PolarArea },
	props: {
		sessions: { type: Array, default: () => [] },
		groupBy: { type: String, default: "loadpoint" },
	},
	data() {
		return {
			textColor: null,
			mutedColor: null,
			gridColor: null,
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
			this.textColor = style.getPropertyValue("--evcc-default-text");
			this.mutedColor = style.getPropertyValue("--bs-gray-medium");
			this.gridColor = style.getPropertyValue("--bs-border-color-translucent");
		},
	},
	computed: {
		chartData() {
			console.log("update chart data");
			const result = {};
			const groups = new Set();

			if (this.sessions.length > 0) {
				const firstDay = new Date(this.sessions[0].created);
				const lastDay = new Date(firstDay.getFullYear(), firstDay.getMonth() + 1, 0);
				const daysInMonth = lastDay.getDate();

				for (let i = 1; i <= daysInMonth; i++) {
					result[i] = {};
				}

				// Populate with actual data
				this.sessions.forEach((session) => {
					const date = new Date(session.created);
					const day = date.getDate();
					const groupKey =
						this.groupBy === "vehicle"
							? session.vehicle || "Unknown"
							: session.loadpoint || "Unknown";
					groups.add(groupKey);

					if (!result[day][groupKey]) {
						result[day][groupKey] = 0;
					}
					result[day][groupKey] += session.chargedEnergy;
				});
			}

			const datasets = Array.from(groups).map((group, i) => ({
				backgroundColor: loadpointColors[i],
				label: group,
				data: Object.values(result).map((day) => day[group] || 0),
			}));

			return {
				labels: Object.keys(result),
				datasets: datasets,
			};
		},
		options() {
			return {
				animations: false,
				locale: this.$i18n?.locale,
				responsive: true,
				maintainAspectRatio: false,
				borderRadius: 5,
				color: this.textColor,
				plugins: {
					legend: {
						position: "bottom",
						align: "center",
						labels: {
							usePointStyle: true,
							pointStyle: "circle",
							padding: 20,
						},
						onClick: function (e, legendItem) {
							const chartInstance = e.chart; // Get the chart instance from the event
							const index = legendItem.datasetIndex;
							const meta = chartInstance.getDatasetMeta(index);

							const highlightAll = meta.highlight === true;
							meta.highlight = !meta.highlight;

							const dimColor = (color) => {
								return color.replace(/FF$/, "20");
							};
							const fullColor = (color) => {
								return color.replace(/20$/, "FF");
							};

							// Loop through all datasets
							chartInstance.data.datasets.forEach((dataset, i) => {
								const datasetMeta = chartInstance.getDatasetMeta(i);
								datasetMeta.hidden = false;
								if (i === index || highlightAll) {
									datasetMeta.highlight = i === index && !highlightAll;
									dataset.backgroundColor = fullColor(dataset.backgroundColor);
								} else {
									datasetMeta.highlight = false;
									dataset.backgroundColor = dimColor(dataset.backgroundColor);
								}
							});

							chartInstance.update();
						},
					},
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
				scales: {
					x: {
						stacked: true,
						border: { display: false },
						grid: { display: false },
						ticks: { color: this.mutedColor },
					},
					y: {
						stacked: true,
						border: { display: false },
						grid: { color: this.gridColor },
						title: {
							text: "kWh",
							display: true,
							color: this.mutedColor,
						},
						ticks: {
							callback: (value, index) =>
								index % 2 === 0 ? this.fmtKWh(value * 1e3, true, false, 0) : null,
							color: this.mutedColor,
						},
						position: "right",
					},
				},
			};
		},
	},
};
</script>
