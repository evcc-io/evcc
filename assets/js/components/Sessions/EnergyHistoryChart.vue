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
import colors, { dimColor, fullColor } from "../../colors";

Chart.register(BarController, BarElement, CategoryScale, LinearScale, Legend, Tooltip);
Chart.defaults.font.family = window
	.getComputedStyle(document.documentElement)
	.getPropertyValue("--bs-font-sans-serif");
Chart.defaults.font.size = 14;
Chart.defaults.layout.padding = 0;
const { generateLabels } = Chart.defaults.plugins.legend.labels;

const GROUPS = {
	SOLAR: "solar",
	LOADPOINT: "loadpoint",
	VEHICLE: "vehicle",
};

export default {
	name: "EnergyHistoryChart",
	components: { Bar },
	props: {
		sessions: { type: Array, default: () => [] },
		groupBy: { type: String, default: GROUPS.SOLAR },
		period: { type: String, default: "total" },
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
	},
	mixins: [formatter],
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
		lastDay() {
			if (this.sessions.length === 0) {
				return null;
			}
			return new Date(this.sessions[this.sessions.length - 1].created);
		},
		chartData() {
			console.log("update history data");
			const result = {};
			const groups = new Set();

			if (this.sessions.length > 0) {
				//const lastDay = new Date(this.year, this.month, 0);
				//const daysInMonth = this.lastDay.getDate();
				let xFrom, xTo;
				if (this.period === "total") {
					xFrom = this.firstDay.getFullYear();
					xTo = this.lastDay.getFullYear();
				} else if (this.period === "year") {
					xFrom = 1;
					xTo = 12;
				} else {
					xFrom = 1;
					xTo = new Date(
						this.lastDay.getFullYear(),
						this.lastDay.getMonth() + 1,
						0
					).getDate();
				}

				// initialize result with empty arrays
				for (let i = xFrom; i <= xTo; i++) {
					result[i] = {};
				}

				// Populate with actual data
				this.sessions.forEach((session) => {
					let index;
					const date = new Date(session.created);
					if (this.period === "month") {
						index = date.getDate();
					} else if (this.period === "year") {
						index = date.getMonth() + 1;
					} else {
						index = date.getFullYear();
					}

					if (this.groupBy === GROUPS.SOLAR) {
						groups.add("grid");
						groups.add("self");
						const charged = session.chargedEnergy;
						const self = (charged / 100) * session.solarPercentage;
						const grid = charged - self;
						result[index].self = (result[index].self || 0) + self;
						result[index].grid = (result[index].grid || 0) + grid;
					} else {
						const groupKey = session[this.groupBy];
						groups.add(groupKey);
						result[index][groupKey] =
							(result[index][groupKey] || 0) + session.chargedEnergy;
					}
				});
			}

			const datasets = Array.from(groups).map((group) => {
				const backgroundColor =
					this.groupBy === GROUPS.SOLAR
						? colors[group]
						: this.colorMappings[this.groupBy][group];
				const label =
					this.groupBy === GROUPS.SOLAR ? this.$t(`sessions.group.${group}`) : group;

				return {
					backgroundColor,
					label,
					data: Object.values(result).map((day) => day[group] || 0),
					borderRadius: (context) => {
						const threshold = 0.04; // 400 Wh
						const { dataIndex, datasetIndex } = context;
						const currentValue = context.dataset.data[dataIndex];
						const previousValuesExist = context.chart.data.datasets
							.slice(datasetIndex + 1)
							.some((dataset) => (dataset?.data[dataIndex] || 0) > threshold);
						return currentValue > threshold && !previousValuesExist
							? { topLeft: 10, topRight: 10 }
							: { topLeft: 0, topRight: 0 };
					},
				};
			});

			return {
				labels: Object.keys(result),
				datasets: datasets,
			};
		},
		options() {
			return {
				locale: this.$i18n?.locale,
				responsive: true,
				maintainAspectRatio: false,
				color: colors.text,
				borderSkipped: false,
				plugins: {
					legend: {
						position: "bottom",
						align: "center",
						labels: {
							usePointStyle: true,
							pointStyle: "circle",
							padding: 20,
							generateLabels: (chart) => {
								const labels = generateLabels(chart);
								labels.forEach((label, datasetIndex) => {
									const sum = chart.data.datasets[datasetIndex].data.reduce(
										(acc, curr) => acc + curr,
										0
									);
									label.text = `${label.text} ${this.fmtKWh(sum * 1e3)}`;
								});
								return labels;
							},
						},
						onClick: function (e, legendItem) {
							const chartInstance = e.chart; // Get the chart instance from the event
							const index = legendItem.datasetIndex;
							const meta = chartInstance.getDatasetMeta(index);

							const highlightAll = meta.highlight === true;
							meta.highlight = !meta.highlight;

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
						positioner: (context) => {
							const tooltip = context.chart.tooltip;
							const tooltipHeight = tooltip.height;
							const tooltipWidth = tooltip.width;
							const windowWidth = window.innerWidth;
							const windowHeight = window.innerHeight;
							const tooltipX = context.tooltipPosition().x;
							const tooltipY = context.tooltipPosition().y;
							const position = {
								x:
									tooltipX + tooltipWidth > windowWidth
										? windowWidth - tooltipWidth
										: tooltipX,
								y:
									tooltipY + tooltipHeight > windowHeight
										? windowHeight - tooltipHeight
										: tooltipY,
							};
							return position;
						},
						callbacks: {
							title: (tooltipItem) => {
								const date = new Date(this.year, this.month, tooltipItem[0].label);
								return this.fmtDayMonth(date);
							},
							label: (tooltipItem) => {
								const datasetLabel = tooltipItem.dataset.label || "";
								const value = tooltipItem.raw || 0;
								return datasetLabel + ": " + this.fmtKWh(value * 1e3);
							},
							labelColor: (item) => {
								const { backgroundColor } = item.element.options;
								const white = "#fff";
								return {
									borderColor: !item.raw ? colors.muted : white,
									backgroundColor,
								};
							},
							labelTextColor: (item) => {
								return !item.raw ? colors.muted : "#fff";
							},
						},
						backgroundColor: "#000",
						itemSort: function (a, b) {
							return b.datasetIndex - a.datasetIndex;
						},
					},
				},
				scales: {
					x: {
						stacked: true,
						border: { display: false },
						grid: { display: false },
						ticks: { color: colors.muted },
					},
					y: {
						stacked: true,
						border: { display: false },
						grid: { color: colors.border },
						title: {
							text: "kWh",
							display: true,
							color: colors.muted,
						},
						ticks: {
							callback: (value, index) =>
								index % 2 === 0 ? this.fmtKWh(value * 1e3, true, false, 0) : null,
							color: colors.muted,
						},
						position: "right",
					},
				},
				animation: false,
			};
		},
	},
};
</script>
