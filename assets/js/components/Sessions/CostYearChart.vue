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
import { RadialLinearScale, PointElement, LineElement, Filler, Tooltip } from "chart.js";
import { registerChartComponents, commonOptions } from "./chartConfig";
import formatter from "../../mixins/formatter";
import colors, { dimColor, lightenColor } from "../../colors";
import LegendList from "./LegendList.vue";
registerChartComponents([RadialLinearScale, PointElement, LineElement, Filler, Tooltip]);

const COST_TYPES = {
	PRICE: "price",
	CO2: "co2",
};

export default {
	name: "CostYearChart",
	components: { Radar, LegendList },
	props: {
		sessions: { type: Array, default: () => [] },
		costType: { type: String, default: COST_TYPES.PRICE },
		currency: { type: String },
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
			console.log("update cost year data");

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
					result[yearString][month] = { avgCost: null };
				}
			}

			// Populate with actual data
			this.sessions.forEach((session) => {
				const date = new Date(session.created);
				const year = `${date.getFullYear()}`;
				const month = `${date.getMonth() + 1}`;

				const value =
					this.costType === COST_TYPES.PRICE
						? session.price
						: session.co2PerKWh * session.chargedEnergy;

				const data = result[year][month];
				data.totalCost = (data.totalCost || 0) + value;
				data.totalKWh = (data.totalKWh || 0) + session.chargedEnergy;
				data.avgCost = data.totalKWh ? data.totalCost / data.totalKWh : null;
			});
			const datasets = years.map((year) => {
				let borderColor =
					this.costType === COST_TYPES.PRICE ? colors.pricePerKWh : colors.co2PerKWh;
				borderColor = lightenColor(borderColor, years.indexOf(year));
				const backgroundColor = years.length === 1 ? dimColor(borderColor) : "transparent";
				const { totalCost, totalKWh } = Object.values(result[year]).reduce(
					(acc, { totalCost = 0, totalKWh = 0 }) => ({
						totalCost: acc.totalCost + totalCost,
						totalKWh: acc.totalKWh + totalKWh,
					}),
					{ totalCost: 0, totalKWh: 0 }
				);
				const yearData = totalKWh ? totalCost / totalKWh : null;
				return {
					backgroundColor,
					borderColor,
					label: year,
					tension: 0.25,
					data: Object.values(result[year]).map(({ avgCost }) => avgCost),
					yearData,
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
					const value =
						dataset.yearData === 0 ? "- %" : this.fmtPricePerKWh(dataset.yearData);
					return {
						label,
						color: dataset.borderColor,
						value,
					};
				});
			} else {
				return this.chartData.labels.map((label, index) => {
					const format = (value) =>
						this.costType === COST_TYPES.PRICE
							? this.fmtPricePerKWh(value, this.currency, true)
							: this.fmtCo2Short(value);

					const value = this.chartData.datasets[0].data[index];
					return {
						label,
						value: value === null ? "-" : format(value),
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
						labelPointStyle: "circle",
						callbacks: {
							label: (tooltipItem) => {
								const value = tooltipItem.raw || 0;
								const datasetLabel = tooltipItem.dataset.label || "";
								return (
									datasetLabel +
									": " +
									this.fmtPricePerKWh(value, this.currency, true)
								);
							},
							labelColor: (item) => {
								const { borderColor } = item.element.options;
								return {
									backgroundColor: borderColor,
								};
							},
						},
					},
				},
				scales: {
					r: {
						beginAtZero: false,
						ticks: {
							color: colors.muted,
							backdropColor: colors.background,
							font: { size: 10 },
							callback: (value) =>
								this.costType === COST_TYPES.PRICE
									? this.fmtPricePerKWh(value, this.currency, true)
									: this.fmtCo2Short(value),
							maxTicksLimit: 4,
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
