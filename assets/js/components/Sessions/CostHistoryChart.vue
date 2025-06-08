<template>
	<div>
		<div style="position: relative; height: 300px" class="my-3">
			<Bar :data="chartData" :options="options" />
		</div>
		<LegendList :legends="legends" />
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { Bar } from "vue-chartjs";
import {
	BarController,
	BarElement,
	CategoryScale,
	Legend,
	LinearScale,
	LineController,
	LineElement,
	Tooltip,
	type ChartData,
	type TooltipItem,
} from "chart.js";
import { registerChartComponents, commonOptions, tooltipLabelColor } from "./chartConfig";
import LegendList from "./LegendList.vue";
import formatter from "@/mixins/formatter";
import colors from "@/colors";
import { TYPES, GROUPS, PERIODS, type Session } from "./types";
import { CURRENCY } from "@/types/evcc";
import type { Context } from "chartjs-plugin-datalabels";

registerChartComponents([
	BarController,
	BarElement,
	CategoryScale,
	Legend,
	LinearScale,
	LineController,
	LineElement,
	Tooltip,
]);

export default defineComponent({
	name: "CostHistoryChart",
	components: { Bar, LegendList },
	mixins: [formatter],
	props: {
		sessions: { type: Array as PropType<Session[]>, default: () => [] },
		groupBy: { type: String as PropType<GROUPS>, default: GROUPS.NONE },
		costType: { type: String as PropType<TYPES>, default: TYPES.PRICE },
		period: { type: String as PropType<PERIODS>, default: PERIODS.TOTAL },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
		suggestedMaxAvgCost: { type: Number, default: 0 },
		suggestedMaxCost: { type: Number, default: 0 },
	},
	computed: {
		firstDay() {
			if (this.sessions.length === 0) {
				return null;
			}
			return new Date(this.sessions[0].created);
		},
		month() {
			return (this.firstDay?.getMonth() || 0) + 1;
		},
		year() {
			return this.firstDay?.getFullYear() || 0;
		},
		lastDay() {
			if (this.sessions.length === 0) {
				return null;
			}
			return new Date(this.sessions[this.sessions.length - 1].created);
		},
		chartData(): ChartData<"bar", number[], unknown> {
			console.log("update cost history data");
			const result: Array<{
				[key: string]: number;
				totalCost: number;
				totalKWh: number;
				avgCost: number;
			}> = [];
			const groups: Set<string> = new Set();

			if (!this.firstDay || !this.lastDay) {
				return { labels: [], datasets: [] };
			}

			if (this.sessions.length > 0) {
				//const lastDay = new Date(this.year, this.month, 0);
				//const daysInMonth = this.lastDay.getDate();
				let xFrom, xTo;
				if (this.period === PERIODS.TOTAL) {
					xFrom = this.firstDay.getFullYear();
					xTo = this.lastDay.getFullYear();
				} else if (this.period === PERIODS.YEAR) {
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
					result[i] = { totalCost: 0, totalKWh: 0, avgCost: 0 };
				}

				// Populate with actual data
				this.sessions.forEach((session) => {
					let index;
					const date = new Date(session.created);
					if (this.period === PERIODS.MONTH) {
						index = date.getDate();
					} else if (this.period === PERIODS.YEAR) {
						index = date.getMonth() + 1;
					} else {
						index = date.getFullYear();
					}

					const groupKey =
						this.groupBy === GROUPS.NONE ? this.costType : session[this.groupBy];
					groups.add(groupKey);

					const value =
						this.costType === TYPES.PRICE
							? session.price || 0
							: (session.co2PerKWh || 0) * (session.chargedEnergy || 0);

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
					type: "bar" as const,
					backgroundColor,
					label,
					data: Object.values(result).map((index) => index[group] || 0),
					borderRadius: (context: Context) => {
						const threshold = 0.04; // 400 Wh
						const { dataIndex, datasetIndex } = context;
						const currentValue = context.dataset.data[dataIndex] as number;
						const previousValuesExist = context.chart.data.datasets
							.filter((dataset) => dataset.type === "bar")
							.slice(datasetIndex + 1)
							.some((dataset: any) => (dataset?.data[dataIndex] || 0) > threshold);
						return (
							currentValue > threshold && !previousValuesExist
								? { topLeft: 10, topRight: 10 }
								: { topLeft: 0, topRight: 0 }
						) as any;
					},
				};
			});

			// add average price line
			const costColor = this.costType === TYPES.PRICE ? colors.pricePerKWh : colors.co2PerKWh;
			datasets.push({
				type: "line" as const,
				label:
					this.costType === TYPES.PRICE
						? this.$t("sessions.avgPrice")
						: this.$t("sessions.co2"),
				data: Object.values(result).map((index) => index.avgCost),
				yAxisID: "y1",
				tension: 0.25,
				pointRadius: 0,
				pointHoverRadius: 6,
				borderColor: costColor,
				backgroundColor: costColor,
				borderWidth: 2,
				spanGaps: true,
			} as any);

			return {
				labels: Object.keys(result),
				datasets: datasets,
			};
		},
		legends() {
			return this.chartData.datasets.map((dataset) => {
				let value = null;

				// line chart handling
				if ((dataset as any).type === "line") {
					const items = dataset.data.filter((v) => v !== null);
					const min = Math.min(...items);
					const max = Math.max(...items);
					const format = (value: number, withUnit: boolean) => {
						return this.costType === TYPES.PRICE
							? this.fmtPricePerKWh(value, this.currency, false, withUnit)
							: withUnit
								? this.fmtCo2Medium(value)
								: this.fmtGrams(value, false);
					};
					value = `${format(min, false)} – ${format(max, true)}`;
				} else {
					const total = dataset.data.reduce((acc, curr) => acc + curr, 0);
					value =
						this.costType === TYPES.PRICE
							? this.fmtMoney(total, this.currency, true, true)
							: this.fmtGrams(total);
				}
				return {
					label: dataset.label || "",
					color: dataset.backgroundColor,
					value,
				};
			});
		},
		options() {
			// capture vue component this to be used in chartjs callbacks
			// eslint-disable-next-line @typescript-eslint/no-this-alias
			const vThis = this;
			return {
				...commonOptions,
				locale: this.$i18n?.locale,
				color: colors.text || "",
				borderSkipped: false,
				maxBarThickness: 40,
				animation: false as const,
				plugins: {
					...commonOptions.plugins,
					tooltip: {
						...commonOptions.plugins.tooltip,
						axis: "x",
						positioner: (context: any) => {
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
							title: (tooltipItem: TooltipItem<"bar">[]) => {
								const { label } = tooltipItem[0];
								if (this.period === PERIODS.TOTAL) {
									return label;
								} else if (this.period === PERIODS.YEAR) {
									const date = new Date(this.year, Number(label) - 1, 1);
									return this.fmtMonth(date, false);
								} else {
									const date = new Date(this.year, this.month - 1, Number(label));
									return this.fmtDayMonth(date);
								}
							},
							label: (tooltipItem: TooltipItem<"bar" | "line">) => {
								const datasetLabel = tooltipItem.dataset.label || "";
								const value = tooltipItem.dataset.data[tooltipItem.dataIndex];

								if (typeof value !== "number") {
									return undefined;
								}

								// line datasets have null values
								if (tooltipItem.dataset.type === "line") {
									const valueFmt =
										this.costType === TYPES.PRICE
											? this.fmtPricePerKWh(value, this.currency, false)
											: this.fmtCo2Medium(value);
									return `${datasetLabel}: ${valueFmt}`;
								}

								return value
									? `${datasetLabel}: ${
											this.costType === TYPES.PRICE
												? this.fmtMoney(value, this.currency, true, true)
												: this.fmtGrams(value)
										}`
									: undefined;
							},
							labelColor: tooltipLabelColor(false),
						},
						itemSort(a: TooltipItem<"bar">, b: TooltipItem<"bar">) {
							return b.datasetIndex - a.datasetIndex;
						},
					},
				},
				scales: {
					x: {
						stacked: true,
						border: { display: false },
						grid: { display: false },
						ticks: {
							color: colors.muted,
							callback(value: number): string {
								return vThis.period === PERIODS.YEAR
									? vThis.fmtMonth(new Date(vThis.year, value, 1), true)
									: (this as any).getLabelForValue(value);
							},
						},
					},
					y: {
						stacked: true,
						position: "right",
						border: { display: false },
						grid: { color: colors.border },
						title: {
							text: "kgCO₂e",
							display: this.costType === TYPES.CO2,
							color: colors.muted,
						},
						ticks: {
							callback: (value: number) =>
								this.costType === TYPES.PRICE
									? this.fmtMoney(value, this.currency, false, true)
									: this.fmtNumber(value / 1e3, 0),
							color: colors.muted,
							maxTicksLimit: 6,
						},
						suggestedMax: this.suggestedMaxCost,
						suggestedMin: 0,
					},
					y1: {
						position: "left",
						border: { display: false },
						suggestedMax: this.suggestedMaxAvgCost,
						grid: {
							drawOnChartArea: false,
						},
						title: {
							text:
								this.costType === TYPES.CO2
									? "gCO₂e/kWh"
									: this.pricePerKWhUnit(this.currency, false),
							display: true,
							color: colors.muted,
						},
						ticks: {
							callback: (value: number) =>
								this.costType === TYPES.PRICE
									? this.fmtPricePerKWh(value, this.currency, false, false)
									: this.fmtNumber(value, 0),
							color: colors.muted,
							maxTicksLimit: 6,
						},
					},
				},
			} as any;
		},
	},
});
</script>
