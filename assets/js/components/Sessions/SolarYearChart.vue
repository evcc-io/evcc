<template>
	<div v-if="chartData.labels.length > 1" class="row">
		<div class="col-12 col-md-6 mb-3">
			<Radar :data="chartData" :options="options" />
		</div>
		<div class="col-12 col-md-6 d-flex align-items-center py-0 py-md-3">
			<LegendList
				:legends="legends"
				:grid="period === 'year'"
				:extra-class="
					period === 'year'
						? 'justify-content-around me-3 flex-row'
						: 'ps-3 flex-md-column'
				"
			/>
		</div>
	</div>
</template>

<script>
import { Radar } from "vue-chartjs";
import { Chart, RadialLinearScale, PointElement, LineElement, Filler, Tooltip } from "chart.js";
import formatter from "../../mixins/formatter";
import colors, { dimColor } from "../../colors";
import LegendList from "./LegendList.vue";

Chart.register(RadialLinearScale, PointElement, LineElement, Filler, Tooltip);
Chart.defaults.font.family = window
	.getComputedStyle(document.documentElement)
	.getPropertyValue("--bs-font-sans-serif");

Chart.defaults.font.size = 14;
Chart.defaults.layout.padding = 0;

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
				const borderColor =
					years.length === 1 ? colors.self : colors.palette[years.indexOf(year)];
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
				locale: this.$i18n?.locale,
				responsive: true,
				aspectRatio: 1,
				maintainAspectRatio: false,
				borderRadius: 8,
				borderWidth: 3,
				color: colors.text,
				spacing: 0,
				plugins: {
					legend: { display: false },
					tooltip: {
						intersect: false,
						mode: "index",
						boxPadding: 5,
						callbacks: {
							label: (tooltipItem) => {
								const value = tooltipItem.raw || 0;
								const datasetLabel = tooltipItem.dataset.label || "";
								return datasetLabel + ": " + this.fmtPercentage(value, 1);
							},
						},
						backgroundColor: "#000",
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
