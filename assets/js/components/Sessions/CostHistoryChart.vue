<template>
	<div>
		<div ref="chartEl" class="chart my-3"></div>
		<LegendList :legends="legends" :device-colors="deviceColors" />
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import {
	axisNameStyle,
	FONT_FAMILY,
	forecastGrid,
	forecastYAxis,
	roundedStackData,
	tooltipStyle,
	tooltipTable,
	xAxisLabelStyle,
	type TooltipRow,
} from "./echarts";
import historyChart from "./historyChart";
import LegendList from "./LegendList.vue";
import colors from "@/colors";
import { TYPES, GROUPS } from "./types";
import { axisScale, type AxisScale } from "@/utils/energyAxis";
import { CURRENCY, type DeviceColors } from "@/types/evcc";

interface Dataset {
	type: "bar" | "line";
	label: string;
	color: string;
	data: (number | null)[];
}

export default defineComponent({
	name: "CostHistoryChart",
	components: { LegendList },
	mixins: [historyChart],
	props: {
		groupBy: { type: String as PropType<GROUPS>, default: GROUPS.NONE },
		costType: { type: String as PropType<TYPES>, default: TYPES.PRICE },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
		deviceColors: { type: Object as PropType<DeviceColors>, default: () => ({}) },
	},
	computed: {
		chartData(): { labels: string[]; datasets: Dataset[] } {
			const result: Array<{
				[key: string]: number;
				totalCost: number;
				totalKWh: number;
				avgCost: number;
			}> = [];
			const groups: Set<string> = new Set();

			const range = this.bucketRange;
			if (!range) {
				return { labels: [], datasets: [] };
			}

			for (let i = range[0]; i <= range[1]; i++) {
				result[i] = { totalCost: 0, totalKWh: 0, avgCost: 0 };
			}

			this.sessions.forEach((session) => {
				const index = this.bucketIndex(new Date(session.created));

				const groupKey =
					this.groupBy === GROUPS.NONE ? this.costType : session[this.groupBy];
				groups.add(groupKey);

				const value =
					this.costType === TYPES.PRICE
						? session.price || 0
						: (session.co2PerKWh || 0) * (session.chargedEnergy || 0);

				const item = result[index]!;
				item[groupKey] = (item[groupKey] || 0) + value;

				item.totalCost = (item.totalCost || 0) + value;
				item.totalKWh = (item.totalKWh || 0) + session.chargedEnergy;
				item.avgCost = item.totalCost / item.totalKWh;
			});

			// stable alphabetical order so entries don't jump between periods
			const sortedGroups = Array.from(groups).sort((a, b) => a.localeCompare(b));

			const datasets: Dataset[] = sortedGroups.map((group) => {
				const colorGroup = this.groupBy === GROUPS.NONE ? "cost" : this.groupBy;
				const color = this.colorMappings[colorGroup][group];
				const label =
					this.groupBy === GROUPS.NONE ? this.$t(`sessions.group.${group}`) : group;
				return {
					type: "bar",
					color,
					label,
					data: Object.values(result).map((index) => index[group] || 0),
				};
			});

			// add average price line
			const costColor =
				(this.costType === TYPES.PRICE ? colors.pricePerKWh : colors.co2PerKWh) || "";
			datasets.push({
				type: "line",
				label:
					this.costType === TYPES.PRICE
						? this.$t("sessions.avgPrice")
						: this.$t("sessions.co2"),
				color: costColor,
				data: Object.values(result).map((index) =>
					index.totalKWh > 0 ? index.avgCost : null
				),
			});

			return {
				labels: Object.keys(result),
				datasets,
			};
		},
		legends() {
			const pickable = this.groupBy !== GROUPS.NONE;
			return this.chartData.datasets.map((dataset) => {
				let value = null;
				let type: "area" | "line" = "area";

				// line chart handling
				if (dataset.type === "line") {
					const items = dataset.data.filter((v): v is number => v !== null);
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
					type = "line";
				} else {
					const total = dataset.data.reduce((acc: number, curr) => acc + (curr || 0), 0);
					value =
						this.costType === TYPES.PRICE
							? this.fmtMoney(total, this.currency, true, true)
							: this.fmtGrams(total);
				}
				return {
					label: dataset.label,
					color: dataset.color,
					value,
					type,
					id: pickable && type !== "line" ? dataset.label || undefined : undefined,
				};
			});
		},
		maxBarTotal() {
			const barDatasets = this.chartData.datasets.filter((d) => d.type === "bar");
			const labelCount = this.chartData.labels.length;
			let max = 0;
			for (let i = 0; i < labelCount; i++) {
				const total = barDatasets.reduce((sum, d) => sum + (d.data[i] || 0), 0);
				if (total > max) max = total;
			}
			return max;
		},
		// CO2 bars are grams
		co2Scale(): AxisScale {
			return axisScale(this.maxBarTotal);
		},
		chartOption(): Record<string, unknown> {
			const { labels, datasets } = this.chartData;
			const head = this.tooltipHead;
			const isPrice = this.costType === TYPES.PRICE;
			const fmtBarValue = (value: number) =>
				isPrice ? this.fmtMoney(value, this.currency, true, true) : this.fmtGrams(value);
			const fmtLineValue = (value: number) =>
				isPrice
					? this.fmtPricePerKWh(value, this.currency, false)
					: this.fmtCo2Medium(value);
			const barDatasets = datasets.filter((d) => d.type === "bar");
			const stackData = roundedStackData(barDatasets, labels.length);
			const line = datasets.find((d) => d.type === "line");
			const series: Record<string, unknown>[] = barDatasets.map((dataset, i) => ({
				name: dataset.label,
				type: "bar",
				stack: "cost",
				barMaxWidth: 40,
				itemStyle: { color: dataset.color },
				data: stackData[i],
			}));
			if (line) {
				series.push({
					name: line.label,
					type: "line",
					yAxisIndex: 1,
					data: line.data,
					smooth: 0.25,
					symbol: "circle",
					symbolSize: 12,
					showSymbol: false,
					connectNulls: true,
					lineStyle: { color: line.color, width: 2 },
					itemStyle: { color: line.color },
				});
			}
			return {
				animation: false,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: { ...forecastGrid(), left: 36, right: 36 },
				tooltip: {
					trigger: "axis",
					axisPointer: { type: "shadow", shadowStyle: { color: "transparent" } },
					...tooltipStyle(colors.text || ""),
					// read values from chartData, rendered series hide tiny top slivers
					formatter: (params: { dataIndex: number }[]) => {
						const idx = params?.[0]?.dataIndex;
						if (idx == null) return "";
						const rows: TooltipRow[] = [];
						// avg line first, then bars top of stack first
						const lineValue = line?.data[idx];
						if (line && lineValue != null) {
							rows.push({ name: line.label, values: [fmtLineValue(lineValue)] });
						}
						[...barDatasets].reverse().forEach((dataset) => {
							const value = dataset.data[idx] || 0;
							if (!value) return;
							rows.push({ name: dataset.label, values: [fmtBarValue(value)] });
						});
						if (!rows.length) return "";
						return tooltipTable(head(labels[idx] ?? ""), rows);
					},
				},
				xAxis: {
					type: "category",
					data: labels,
					axisLine: { show: false },
					axisTick: { show: false },
					splitLine: { show: false },
					axisLabel: {
						...xAxisLabelStyle(),
						formatter: this.xAxisLabel,
					},
				},
				yAxis: [
					forecastYAxis({
						position: "right",
						min: 0,
						splitLine: {
							showMinLine: true,
							showMaxLine: true,
							lineStyle: { color: colors.border || "" },
						},
						...(this.costType === TYPES.CO2
							? {
									name: this.co2Scale.small ? "g" : "kg",
									...(this.co2Scale.small
										? {
												max: this.co2Scale.limit,
												interval: this.co2Scale.limit / 4,
											}
										: {}),
								}
							: {}),
						...axisNameStyle(),
						axisLabel: {
							color: colors.muted || "",
							hideOverlap: true,
							formatter: (value: number) =>
								isPrice
									? this.fmtMoney(
											value,
											this.currency,
											this.maxBarTotal < 4,
											true
										)
									: this.co2Scale.small
										? this.fmtNumber(value, 0)
										: this.fmtNumber(value / 1e3, this.co2Scale.digits),
						},
					}),
					forecastYAxis({
						position: "left",
						splitLine: { show: false },
						name: isPrice ? this.pricePerKWhUnit(this.currency, false) : "g/kWh",
						...axisNameStyle("right"),
						axisLabel: {
							color: colors.muted || "",
							hideOverlap: true,
							formatter: (value: number) =>
								isPrice
									? this.fmtPricePerKWh(value, this.currency, false, false)
									: this.fmtNumber(value, 0),
						},
					}),
				],
				series,
			};
		},
	},
});
</script>

<style scoped>
.chart {
	width: 100%;
	height: 300px;
}
</style>
