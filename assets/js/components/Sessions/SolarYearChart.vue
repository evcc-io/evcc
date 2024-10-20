<template>
	<div v-if="chartData.labels.length > 1" class="row">
		<div class="col-12 col-md-6 mb-3">
			<Radar :data="chartData" :options="options" />
		</div>
		<div class="col-12 col-md-6 d-flex align-items-center py-0 py-md-3">
			<LegendList :legends="legends" small-equal-widths />
		</div>
	</div>
</template>

<script>
import { Radar } from "vue-chartjs";
import { RadialLinearScale, PointElement, LineElement, Filler, Tooltip } from "chart.js";
import { registerChartComponents, commonOptions, tooltipLabelColor } from "./chartConfig";
import formatter from "../../mixins/formatter";
import colors, { dimColor } from "../../colors";
import LegendList from "./LegendList.vue";

registerChartComponents([RadialLinearScale, PointElement, LineElement, Filler, Tooltip]);

export default {
	name: "SolarYearChart",
	components: { Radar, LegendList },
	props: {
		sessions: { type: Array, default: () => [] },
		period: { type: String, default: "total" },
	},
	mixins: [formatter],
	computed: {
		firstDay() {
			if (this.sessions.length === 0) {
				return null;
			}
			return new Date(this.sessions[0].created);
		},
		lastDay() {
			if (this.sessions.length === 0) {
				return null;
			}
			return new Date(this.sessions[this.sessions.length - 1].created);
		},
		chartData() {
			console.log("update solar month data");

			if (!this.sessions.length > 0) return { labels: [], datasets: [] };

			const firstYear = this.firstDay.getFullYear();
			const lastYear = this.lastDay.getFullYear();

			const result = {};

			const years = [];

			// initialize result for years and months
			for (let year = lastYear; year >= firstYear; year--) {
				const yearString = `${year}`;
				years.push(yearString);
				result[yearString] = {};
				console.log("year", yearString);

				for (let month = 1; month <= 12; month++) {
					result[yearString][month] = {};
				}
			}

			// Populate with actual data
			this.sessions.forEach((session) => {
				const date = new Date(session.created);
				const year = `${date.getFullYear()}`;
				const month = `${date.getMonth() + 1}`;

				const charged = session.chargedEnergy;
				const self = (charged / 100) * session.solarPercentage;
				const grid = charged - self;
				result[year][month].self = (result[year][month].self || 0) + self;
				result[year][month].grid = (result[year][month].grid || 0) + grid;
			});

			const datasets = years.map((year) => {
				const borderColor = colors.selfPalette[years.indexOf(year)];
				const backgroundColor = years.length === 1 ? dimColor(borderColor) : "transparent";
				return {
					backgroundColor,
					borderColor,
					label: year,
					data: Object.values(result[year]).map(({ self = 0, grid = 0 }) => {
						const total = self + grid;
						return total === 0 ? null : (self / total) * 100;
					}),
					yearData: Object.values(result[year]).reduce(
						(acc, { self = 0, grid = 0 }) => ({
							self: acc.self + self,
							grid: acc.grid + grid,
						}),
						{ self: 0, grid: 0 }
					),
				};
			});

			const labels = Object.keys(result[firstYear]).map((month) =>
				this.fmtMonth(new Date(firstYear, month - 1, 1), true)
			);

			return {
				labels,
				datasets,
			};
		},
		legends() {
			if (this.period === "total") {
				return this.chartData.datasets.map((dataset) => {
					const label = dataset.label;
					const { self, grid } = dataset.yearData;
					const total = self + grid;
					const value = total === 0 ? "- %" : this.fmtPercentage((self / total) * 100, 1);
					return {
						label,
						color: dataset.borderColor,
						value,
					};
				});
			} else {
				return this.chartData.labels.map((label, index) => {
					const value = this.chartData.datasets[0].data[index];
					return {
						label,
						value:
							value === null
								? "- %"
								: this.fmtPercentage(this.chartData.datasets[0].data[index], 1),
					};
				});
			}
		},
		options() {
			return {
				...commonOptions,
				locale: this.$i18n?.locale,
				aspectRatio: 1,
				borderWidth: 4,
				color: colors.text,
				spacing: 0,
				radius: "100%",
				elements: { line: { tension: 0.05 } },
				plugins: {
					...commonOptions.plugins,
					tooltip: {
						...commonOptions.plugins.tooltip,
						intersect: false,
						mode: "index",
						position: "topBottomCenter",
						callbacks: {
							label: (tooltipItem) => {
								const value = tooltipItem.raw || 0;
								const datasetLabel = tooltipItem.dataset.label || "";
								return datasetLabel + ": " + this.fmtPercentage(value, 1);
							},
							labelColor: tooltipLabelColor(true),
						},
					},
				},
				scales: {
					r: {
						min: 0,
						max: 100,
						beginAtZero: false,
						ticks: {
							stepSize: 20,
							color: colors.muted,
							backdropColor: colors.background,
							font: { size: 10 },
							callback: (value) => this.fmtPercentage(value, 0),
						},
						angleLines: { display: false },
						grid: { color: colors.border },
					},
				},
			};
		},
	},
};
</script>
