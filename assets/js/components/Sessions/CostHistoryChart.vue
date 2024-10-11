<template>
	<div>
		<div style="position: relative; height: 300px" class="my-3">
			<Bar :data="chartData" :options="options" />
		</div>
		<LegendList :legends="legends" />
	</div>
</template>

<script>
import { Bar } from "vue-chartjs";
import { BarController, BarElement, CategoryScale, LinearScale, Legend, Tooltip } from "chart.js";
import { registerChartComponents, commonOptions } from "./chartConfig";
import LegendList from "./LegendList.vue";
import formatter, { POWER_UNIT } from "../../mixins/formatter";
import colors from "../../colors";

registerChartComponents([BarController, BarElement, CategoryScale, LinearScale, Legend, Tooltip]);

const GROUPS = {
	NONE: "none",
	LOADPOINT: "loadpoint",
	VEHICLE: "vehicle",
};

const COST_TYPES = {
	PRICE: "price",
	CO2: "co2",
};

export default {
	name: "CostHistoryChart",
	components: { Bar, LegendList },
	props: {
		sessions: { type: Array, default: () => [] },
		groupBy: { type: String, default: GROUPS.NONE },
		costType: { type: String, default: COST_TYPES.PRICE },
		period: { type: String, default: "total" },
		currency: { type: String, default: "EUR" },
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

					const groupKey =
						this.groupBy === GROUPS.NONE ? this.costType : session[this.groupBy];
					groups.add(groupKey);

					const value =
						this.costType === COST_TYPES.PRICE
							? session.price
							: session.co2PerKWh * session.chargedEnergy;

					result[index][groupKey] = (result[index][groupKey] || 0) + value;

					result[index].totalCost = (result[index].totalCost || 0) + value;
					result[index].totalKWh = (result[index].totalKWh || 0) + session.chargedEnergy;
					result[index].avgCost = result[index].totalCost / result[index].totalKWh;
				});
			}

			const datasets = Array.from(groups).map((group) => {
				const colorGroup = this.groupBy === GROUPS.NONE ? "cost" : this.groupBy;
				const backgroundColor = this.colorMappings[colorGroup][group];
				const label =
					this.groupBy === GROUPS.NONE ? this.$t(`sessions.group.${group}`) : group;

				return {
					type: "bar",
					backgroundColor,
					label,
					data: Object.values(result).map((index) => index[group] || 0),
					borderRadius: (context) => {
						const threshold = 0.04; // 400 Wh
						const { dataIndex, datasetIndex } = context;
						const currentValue = context.dataset.data[dataIndex];
						const previousValuesExist = context.chart.data.datasets
							.filter((dataset) => dataset.type === "bar")
							.slice(datasetIndex + 1)
							.some((dataset) => (dataset?.data[dataIndex] || 0) > threshold);
						return currentValue > threshold && !previousValuesExist
							? { topLeft: 10, topRight: 10 }
							: { topLeft: 0, topRight: 0 };
					},
				};
			});

			// add average price line
			const costColor =
				this.costType === COST_TYPES.PRICE ? colors.pricePerKWh : colors.co2PerKWh;
			datasets.push({
				type: "line",
				label:
					this.costType === COST_TYPES.PRICE
						? this.pricePerKWhUnit(this.currency, false)
						: "gCO₂e/kWh",
				data: Object.values(result).map((index) => index.avgCost || null),
				yAxisID: "y1",
				tension: 0.25,
				pointRadius: 0,
				pointHoverRadius: 6,
				borderColor: costColor,
				backgroundColor: costColor,
				borderWidth: 2,
				spanGaps: true,
			});

			return {
				labels: Object.keys(result),
				datasets: datasets,
			};
		},
		legends() {
			return this.chartData.datasets.map((dataset) => {
				let value = null;

				// line chart handling
				if (dataset.type === "line") {
					const items = dataset.data.filter((v) => v !== null);
					const min = Math.min(...items);
					const max = Math.max(...items);
					const format = (value, withUnit) => {
						return this.costType === COST_TYPES.PRICE
							? this.fmtPricePerKWh(value, this.currency, true, withUnit)
							: this.fmtGrams(value, withUnit);
					};
					value = `${format(min, false)} – ${format(max, true)}`;
				} else {
					const total = dataset.data.reduce((acc, curr) => acc + curr, 0);
					value =
						this.costType === COST_TYPES.PRICE
							? this.fmtMoney(total, this.currency, true, true)
							: this.fmtGrams(total);
				}
				return {
					label: dataset.label,
					color: dataset.backgroundColor,
					value,
				};
			});
		},
		options() {
			return {
				...commonOptions,
				locale: this.$i18n?.locale,
				color: colors.text,
				borderSkipped: false,
				maxBarThickness: 40,
				animation: false,
				plugins: {
					...commonOptions.plugins,
					tooltip: {
						...commonOptions.plugins.tooltip,
						mode: "index",
						intersect: false,
						positioner: (context) => {
							const { chart, tooltipPosition } = context;
							const { tooltip } = chart;
							const { width, height } = tooltip;
							const { x, y } = tooltipPosition();
							const { innerWidth, innerHeight } = window;

							return {
								x: Math.min(x, innerWidth - width),
								y: Math.min(y, innerHeight - height),
							};
						},
						callbacks: {
							title: (tooltipItem) => {
								const { label } = tooltipItem[0];
								if (this.period === "total") {
									return label;
								} else if (this.period === "year") {
									const date = new Date(this.year, label - 1, 1);
									return this.fmtMonth(date);
								} else {
									const date = new Date(this.year, this.month, label);
									return this.fmtDayMonth(date);
								}
							},
							label: (tooltipItem) => {
								const datasetLabel = tooltipItem.dataset.label || "";
								const value = tooltipItem.raw;

								// line datasets have null values
								if (tooltipItem.dataset.type === "line") {
									if (value === null) {
										return null;
									}

									return (
										"⌀ " +
										(this.costType === COST_TYPES.PRICE
											? this.fmtPricePerKWh(value, this.currency, false)
											: this.fmtCo2Medium(value))
									);
								}

								return (
									datasetLabel +
									": " +
									(this.costType === COST_TYPES.PRICE
										? this.fmtMoney(value || 0, this.currency, true, true)
										: this.fmtGrams(value || 0))
								);
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
						position: "right",
						border: { display: false },
						grid: { color: colors.border },
						title: {
							text: "kgCO₂e",
							display: this.costType === COST_TYPES.CO2,
							color: colors.muted,
						},
						ticks: {
							callback: (value) =>
								this.costType === COST_TYPES.PRICE
									? this.fmtMoney(value, this.currency, false, true)
									: this.fmtNumber(value / 1e3, 0),
							color: colors.muted,
							maxTicksLimit: 6,
						},
						min: 0,
					},
					y1: {
						position: "left",
						border: { display: false },
						grid: {
							drawOnChartArea: false,
						},
						title: {
							text:
								this.costType === COST_TYPES.CO2
									? "gCO₂e/kWh"
									: this.pricePerKWhUnit(this.currency, false),
							display: true,
							color: colors.muted,
						},
						ticks: {
							callback: (value) =>
								this.costType === COST_TYPES.PRICE
									? this.fmtPricePerKWh(value, this.currency, false, false)
									: this.fmtNumber(value, 0),
							color: colors.muted,
							maxTicksLimit: 6,
						},
					},
				},
			};
		},
	},
};
</script>
