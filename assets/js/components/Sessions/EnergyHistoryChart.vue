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
	LinearScale,
	Legend,
	Tooltip,
	type ChartData,
	type TooltipModel,
	type TooltipItem,
} from "chart.js";
import { registerChartComponents, commonOptions, tooltipLabelColor } from "./chartConfig";
import LegendList from "./LegendList.vue";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import colors from "@/colors";
import { GROUPS, PERIODS, type Session } from "./types";
import type { Context } from "chartjs-plugin-datalabels";

registerChartComponents([BarController, BarElement, CategoryScale, LinearScale, Legend, Tooltip]);

export default defineComponent({
	name: "EnergyHistoryChart",
	components: { Bar, LegendList },
	mixins: [formatter],
	props: {
		sessions: { type: Array as PropType<Session[]>, default: () => [] },
		groupBy: { type: String as PropType<GROUPS>, default: GROUPS.NONE },
		period: { type: String as PropType<PERIODS>, default: PERIODS.TOTAL },
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
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
			console.log("update energy history data");
			const result: Record<number, Record<string, number>> = {};
			const groups: Set<string> = new Set();

			if (this.firstDay && this.lastDay) {
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
					result[i] = {};
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

					if (this.groupBy === GROUPS.NONE) {
						groups.add("grid");
						groups.add("self");
						const charged = session.chargedEnergy;
						const self = (charged / 100) * session.solarPercentage;
						const grid = charged - self;
						result[index]["self"] = (result[index]["self"] || 0) + self;
						result[index]["grid"] = (result[index]["grid"] || 0) + grid;
					} else {
						const groupKey = session[this.groupBy];
						groups.add(groupKey);
						result[index][groupKey] =
							(result[index][groupKey] || 0) + session.chargedEnergy;
					}
				});
			}

			const datasets = Array.from(groups).map((group) => {
				const colorGroup = this.groupBy === GROUPS.NONE ? "solar" : this.groupBy;
				const backgroundColor = this.colorMappings[colorGroup][group];
				const label =
					this.groupBy === GROUPS.NONE ? this.$t(`sessions.group.${group}`) : group;

				return {
					backgroundColor,
					label,
					data: Object.values(result).map((day) => day[group] || 0),
					borderRadius: (context: Context) => {
						const threshold = 0.04; // 400 Wh
						const { dataIndex, datasetIndex } = context;
						const currentValue = context.dataset.data[dataIndex] as number;
						const previousValuesExist = context.chart.data.datasets
							.slice(datasetIndex + 1)
							.some((dataset: any) => (dataset?.data[dataIndex] || 0) > threshold);
						return currentValue > threshold && !previousValuesExist
							? { topLeft: 10, topRight: 10, bottomLeft: 0, bottomRight: 0 }
							: { topLeft: 0, topRight: 0, bottomLeft: 0, bottomRight: 0 };
					},
				};
			});

			return {
				labels: Object.keys(result),
				datasets: datasets,
			};
		},
		legends() {
			return this.chartData.datasets.map((dataset) => ({
				label: dataset.label || "",
				color: dataset.backgroundColor,
				value: this.fmtWh(
					dataset.data.reduce((acc, curr) => acc + curr, 0) * 1e3,
					POWER_UNIT.AUTO
				),
			}));
		},
		options() {
			// capture vue component this to be used in chartjs callbacks
			// eslint-disable-next-line @typescript-eslint/no-this-alias
			const vThis = this;
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
						axis: "x",
						positioner: (context: TooltipModel<"bar">) => {
							const { chart, tooltipPosition } = context;
							const { tooltip } = chart;
							const { width, height } = tooltip || {};
							const { x, y } = tooltipPosition(false);
							const { innerWidth, innerHeight } = window;

							return {
								x: Math.min(x, innerWidth - (width || 0)),
								y: Math.min(y, innerHeight - (height || 0)),
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
							label: (tooltipItem: TooltipItem<"bar">) => {
								const datasetLabel = tooltipItem.dataset.label || "";
								const value =
									(tooltipItem.dataset.data[tooltipItem.dataIndex] as number) ||
									0;
								return value
									? `${datasetLabel}: ${this.fmtWh(value * 1e3, POWER_UNIT.AUTO)}`
									: null;
							},
							labelColor: tooltipLabelColor(false),
							labelPointStyle() {
								return {
									pointStyle: "circle",
								};
							},
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
							callback(value: number) {
								return vThis.period === PERIODS.YEAR
									? vThis.fmtMonth(new Date(vThis.year, value, 1), true)
									: (this as any).getLabelForValue(value);
							},
						},
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
							callback: (value: number) =>
								this.fmtWh(value * 1e3, POWER_UNIT.KW, false, 0),
							color: colors.muted,
							maxTicksLimit: 6,
						},
						position: "right",
						min: 0,
					},
				},
			} as any;
		},
	},
});
</script>
