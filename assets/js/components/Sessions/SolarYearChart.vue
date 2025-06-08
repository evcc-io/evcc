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

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { Radar } from "vue-chartjs";
import {
	RadialLinearScale,
	PointElement,
	LineElement,
	Filler,
	Tooltip,
	type TooltipItem,
} from "chart.js";
import { registerChartComponents, commonOptions, tooltipLabelColor } from "./chartConfig.ts";
import formatter from "@/mixins/formatter";
import colors, { dimColor } from "@/colors";
import LegendList from "./LegendList.vue";
import type { Legend, PERIODS, Session } from "./types.ts";

registerChartComponents([RadialLinearScale, PointElement, LineElement, Filler, Tooltip]);

export default defineComponent({
	name: "SolarYearChart",
	components: { Radar, LegendList },
	mixins: [formatter],
	props: {
		sessions: { type: Array as PropType<Session[]>, default: () => [] },
		period: { type: String as PropType<PERIODS>, default: "total" },
	},
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

			if (!this.firstDay || !this.lastDay) {
				return { labels: [], datasets: [] };
			}

			const firstYear = this.firstDay.getFullYear();
			const lastYear = this.lastDay.getFullYear();

			const result: Record<string, Record<string, { self: number; grid: number }>> = {};

			const years: string[] = [];

			// initialize result for years and months
			for (let year = lastYear; year >= firstYear; year--) {
				const yearString = `${year}`;
				years.push(yearString);
				result[yearString] = {};
				console.log("year", yearString);

				for (let month = 1; month <= 12; month++) {
					result[yearString][month] = { self: 0, grid: 0 };
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
				result[year][month].self += self;
				result[year][month].grid += grid;
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
				this.fmtMonth(new Date(firstYear, parseInt(month) - 1, 1), true)
			);

			return {
				labels,
				datasets,
			};
		},
		legends(): Legend[] {
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
						color: null,
						value:
							value === null
								? "- %"
								: this.fmtPercentage(
										this.chartData.datasets[0].data[index] || 0,
										1
									),
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
				color: colors.text || "",
				spacing: 0,
				radius: "100%",
				elements: { line: { tension: 0.05 } },
				plugins: {
					...commonOptions.plugins,
					tooltip: {
						...commonOptions.plugins.tooltip,
						axis: "xy",
						position: "topBottomCenter",
						callbacks: {
							label: (tooltipItem: TooltipItem<"radar">) => {
								const value = tooltipItem.dataset.data[tooltipItem.dataIndex] || 0;
								const datasetLabel = tooltipItem.dataset.label || "";
								return datasetLabel + ": " + this.fmtPercentage(value, 1);
							},
							labelColor: tooltipLabelColor(true),
						},
					} as any,
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
							callback: (value: number) => this.fmtPercentage(value, 0),
						},
						angleLines: { display: false },
						grid: { color: colors.border },
					},
				},
			} as any;
		},
	},
});
</script>
