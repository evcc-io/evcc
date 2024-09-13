<template>
	<div style="position: relative; height: 300px">
		<Bar :data="chartData" :options="options" />
	</div>
</template>

<script>
import { Bar } from "vue-chartjs";
import {
	Chart,
	BarController,
	BarElement,
	CategoryScale,
	LinearScale,
	Legend,
	Tooltip,
} from "chart.js";
import formatter from "../../mixins/formatter";

Chart.register(BarController, BarElement, CategoryScale, LinearScale, Legend, Tooltip);
Chart.defaults.font.family = window
	.getComputedStyle(document.documentElement)
	.getPropertyValue("--bs-font-sans-serif");

Chart.defaults.font.size = 14;
Chart.defaults.layout.padding = 0;
Chart.defaults.animation = false;

export default {
	name: "EnergyHistoryChart",
	components: { Bar },
	props: {
		sessions: { type: Array, default: () => [] },
		groupBy: { type: String, default: "loadpoint" },
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
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
		firstDay() {
			if (this.sessions.length === 0) {
				return null;
			}
			return new Date(this.sessions[0].created);
		},
		month() {
			return this.firstDay?.getMonth() + 1;
		},
		year() {
			return this.firstDay?.getFullYear();
		},
		chartData() {
			console.log("update chart data");
			const result = {};
			const groups = new Set();

			if (this.sessions.length > 0) {
				const lastDay = new Date(this.year, this.month, 0);
				const daysInMonth = lastDay.getDate();

				for (let i = 1; i <= daysInMonth; i++) {
					result[i] = {};
				}
				// Populate with actual data
				this.sessions.forEach((session) => {
					const day = new Date(session.created).getDate();
					const groupKey = session[this.groupBy];
					groups.add(groupKey);
					if (!result[day][groupKey]) {
						result[day][groupKey] = 0;
					}
					result[day][groupKey] += session.chargedEnergy;
				});
			}

			const datasets = Array.from(groups).map((group) => ({
				backgroundColor: this.colorMappings[this.groupBy][group],
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
				borderRadius: 6,
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
							title: (tooltipItem) => {
								const date = new Date(this.year, this.month, tooltipItem[0].label);
								return this.fmtDayMonth(date);
							},
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
