<template>
	<div v-if="chartData.labels.length > 1" class="row">
		<div class="col-12 col-md-6 mb-3">
			<Radar :data="chartData" :options="options" />
		</div>
		<div class="col-12 col-md-6 d-flex align-items-center py-0 py-md-3">
			<LegendList :legends="legends" extra-class="flex-md-column" />
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
	name: "SolarMonthChart",
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
			const result = {};

			if (this.sessions.length > 0) {
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

					const charged = session.chargedEnergy;
					const self = (charged / 100) * session.solarPercentage;
					const grid = charged - self;
					result[index].self = (result[index].self || 0) + self;
					result[index].grid = (result[index].grid || 0) + grid;
				});
			}

			const dataset = {
				backgroundColor: dimColor(colors.self),
				borderColor: colors.self,
				label: this.$t("sessions.group.self"),
				data: Object.values(result).map(({ self = 0, grid = 0 }) => {
					const total = self + grid;
					return total === 0 ? null : (self / total) * 100;
				}),
			};
			console.log(dataset);

			return {
				labels: Object.keys(result).map((key) =>
					this.fmtMonth(new Date(this.firstDay.getFullYear(), key - 1, 1), true)
				),
				datasets: [dataset],
			};
		},
		legends() {
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
					tooltip: { enabled: false },
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
