<template>
	<div>
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
	Colors,
} from "chart.js";
import formatter from "../../mixins/formatter";

Chart.register(BarController, BarElement, CategoryScale, LinearScale, Legend, Colors);

export default {
	name: "Chart",
	components: { Bar },
	props: {
		sessions: { type: Array, default: () => [] },
	},
	mixins: [formatter],
	computed: {
		chartData() {
			console.log("update chart data");
			const result = {};
			const loadpoints = new Set();

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
					const loadpoint = session.loadpoint || "Unknown";
					loadpoints.add(loadpoint);

					if (!result[day][loadpoint]) {
						result[day][loadpoint] = 0;
					}
					result[day][loadpoint] += session.chargedEnergy;
				});
			}

			const datasets = Array.from(loadpoints).map((loadpoint) => ({
				label: loadpoint,
				data: Object.values(result).map((day) => day[loadpoint] || 0),
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
				scales: {
					x: {
						stacked: true,
					},
					y: {
						stacked: true,
						ticks: {
							callback: (value) => this.fmtKWh(value * 1e3, true, true, 0),
						},
						position: "right",
					},
				},
			};
		},
	},
};
</script>
