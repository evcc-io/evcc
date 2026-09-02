<template>
	<div v-if="chartData.labels.length > 1" class="row">
		<div class="col-12 col-md-6 col-lg-12 col-xxl-6 mb-3">
			<div ref="chartEl" class="round-chart"></div>
		</div>
		<div class="col-12 col-md-6 col-lg-12 col-xxl-6 d-flex align-items-center">
			<LegendList :legends="legends" small-equal-widths />
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import {
	FONT_FAMILY,
	topBottomCenterPosition,
	tooltipStyle,
	tooltipTable,
	type TooltipRow,
	lineDefaults,
} from "./echarts";
import echartsChart from "@/mixins/echartsChart";
import formatter from "@/mixins/formatter";
import colors, { dimColor } from "@/colors";
import LegendList from "./LegendList.vue";
import type { Legend, PERIODS, Session } from "./types.ts";

interface Dataset {
	label: string;
	borderColor: string | undefined;
	data: (number | null)[];
	yearData: { self: number; grid: number };
}

export default defineComponent({
	name: "SolarYearChart",
	components: { LegendList },
	mixins: [formatter, echartsChart],
	props: {
		sessions: { type: Array as PropType<Session[]>, default: () => [] },
		period: { type: String as PropType<PERIODS>, default: "total" },
	},
	computed: {
		firstDay() {
			if (this.sessions.length === 0) {
				return null;
			}
			return new Date(this.sessions[0]!.created);
		},
		lastDay() {
			if (this.sessions.length === 0) {
				return null;
			}
			return new Date(this.sessions[this.sessions.length - 1]!.created);
		},
		chartData(): { labels: string[]; datasets: Dataset[] } {
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
				const monthData = result[year]![month]!;
				monthData.self += self;
				monthData.grid += grid;
			});

			const datasets = years.map((year) => {
				const borderColor = colors.selfPalette[years.indexOf(year)] || undefined;
				return {
					borderColor,
					label: year,
					data: Object.values(result[year] || {}).map(({ self = 0, grid = 0 }) => {
						const total = self + grid;
						return total === 0 ? null : (self / total) * 100;
					}),
					yearData: Object.values(result[year] || {}).reduce(
						(acc, { self = 0, grid = 0 }) => ({
							self: acc.self + self,
							grid: acc.grid + grid,
						}),
						{ self: 0, grid: 0 }
					),
				};
			});

			const labels = Object.keys(result[firstYear] || {}).map((month) =>
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
				const dataset = this.chartData.datasets[0]!;
				return this.chartData.labels.map((label, index) => {
					const value = dataset.data[index];
					return {
						label,
						color: null,
						value:
							value === null ? "- %" : this.fmtPercentage((value as number) || 0, 1),
					};
				});
			}
		},
		chartOption(): Record<string, unknown> {
			const { labels, datasets } = this.chartData;
			const singleYear = datasets.length === 1;
			return {
				animation: false,
				textStyle: { fontFamily: FONT_FAMILY },
				tooltip: {
					trigger: "item",
					...tooltipStyle(colors.text || ""),
					position: topBottomCenterPosition(() => this.chart),
					formatter: (params: { name: string; value: (number | null)[] }) => {
						const rows: TooltipRow[] = [];
						(params.value || []).forEach((value, index) => {
							if (value === null) return;
							rows.push({
								name: labels[index] ?? "",
								values: [this.fmtPercentage(value, 1)],
							});
						});
						return tooltipTable(params.name, rows);
					},
				},
				radar: {
					// tick labels on the first spoke only
					indicator: labels.map((name, i) => ({
						name,
						min: 0,
						max: 100,
						...(i === 0
							? {
									axisLabel: {
										show: true,
										fontSize: 10,
										color: colors.muted || "",
										backgroundColor: colors.box || "",
										padding: [1, 3],
										formatter: (value: number) => this.fmtPercentage(value, 0),
									},
								}
							: {}),
					})),
					radius: "72%",
					axisNameGap: 12,
					splitNumber: 5,
					axisName: { color: colors.muted || "", fontSize: 14 },
					axisLine: { show: false },
					splitLine: { lineStyle: { color: colors.border || "" } },
					splitArea: { show: false },
				},
				series: [
					{
						type: "radar",
						symbol: "none",
						data: datasets.map((dataset) => ({
							name: dataset.label,
							value: dataset.data,
							lineStyle: { color: dataset.borderColor, ...lineDefaults },
							itemStyle: { color: dataset.borderColor },
							areaStyle:
								singleYear && dataset.borderColor
									? { color: dimColor(dataset.borderColor) }
									: undefined,
						})),
					},
				],
			};
		},
	},
});
</script>
