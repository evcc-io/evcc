<template>
	<div>
		<div class="overflow-x-auto overflow-x-md-auto chart-container">
			<div style="position: relative; height: 220px" class="chart">
				<Bar :data="chartData" :options="options" />
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { Bar } from "vue-chartjs";
import {
	BarController,
	BarElement,
	LineController,
	LineElement,
	LinearScale,
	TimeSeriesScale,
	Legend,
	Tooltip,
	PointElement,
	Filler,
} from "chart.js";
import ChartDataLabels from "chartjs-plugin-datalabels";
import "chartjs-adapter-dayjs-4/dist/chartjs-adapter-dayjs-4.esm";
import { registerChartComponents, commonOptions } from "./Sessions/chartConfig";
import formatter, { POWER_UNIT } from "../mixins/formatter";
import colors, { lighterColor } from "../colors";
import { energyByDay, highestSlotIndexByDay, type PriceSlot } from "../utils/forecast";

registerChartComponents([
	BarController,
	BarElement,
	LineController,
	LineElement,
	Filler,
	LinearScale,
	TimeSeriesScale,
	Legend,
	Tooltip,
	PointElement,
	ChartDataLabels,
]);

export default defineComponent({
	name: "ForecastChart",
	components: { Bar },
	mixins: [formatter],
	props: {
		grid: { type: Array as PropType<PriceSlot[]> },
		solar: { type: Array as PropType<PriceSlot[]> },
		co2: { type: Array as PropType<PriceSlot[]> },
		currency: { type: String as PropType<string> },
	},
	computed: {
		startDate() {
			const now = new Date();
			const slots = this.grid || this.co2 || this.solar || [];
			const currentSlot = slots.find(({ start, end }) => {
				return new Date(start) <= now && new Date(end) > now;
			});
			if (currentSlot) {
				return new Date(currentSlot.start);
			}
			return now;
		},
		solarSlots() {
			return this.filterSlots(this.solar);
		},
		gridSlots() {
			return this.filterSlots(this.grid);
		},
		maxPriceIndex() {
			return this.gridSlots.reduce((max, slot, index) => {
				return slot.price > this.gridSlots[max].price ? index : max;
			}, 0);
		},
		minPriceIndex() {
			return this.gridSlots.reduce((min, slot, index) => {
				return slot.price < this.gridSlots[min].price ? index : min;
			}, 0);
		},
		solarHighlights() {
			return [0, 1, 2].map((day) => {
				const energy = energyByDay(this.solarSlots, day);
				const index = highestSlotIndexByDay(this.solarSlots, day);
				return { index, energy };
			});
		},
		chartData() {
			const datasets: unknown[] = [];
			if (this.solarSlots.length > 0) {
				datasets.push({
					label: "solar",
					type: "line",
					data: this.solarSlots.map((slot, index) => ({
						y: slot.price,
						x: new Date(slot.start),
						highlight: this.solarHighlights.find(({ index: i }) => i === index)?.energy,
					})),
					yAxisID: "yForecast",
					backgroundColor: lighterColor(colors.self),
					borderColor: colors.self,
					fill: "origin",
					tension: 0.5,
					pointRadius: 0,
					spanGaps: true,
				});
			}
			if (this.gridSlots.length > 0) {
				datasets.push({
					label: "price",
					data: this.gridSlots.map((slot, index) => ({
						y: slot.price,
						x: new Date(slot.start),
						highlight: index === this.maxPriceIndex || index === this.minPriceIndex,
					})),
					yAxisID: "yPrice",
					borderRadius: 8,
					backgroundColor: colors.light,
					borderColor: colors.light,
				});
			}
			return {
				datasets,
			};
		},
		options() {
			// eslint-disable-next-line @typescript-eslint/no-this-alias
			const vThis = this;
			return {
				...commonOptions,
				locale: this.$i18n?.locale,
				layout: { padding: { top: 32 } },
				color: colors.text,
				borderSkipped: false,
				animation: false,
				categoryPercentage: 0.7,
				plugins: {
					...commonOptions.plugins,
					datalabels: {
						backgroundColor: function (context) {
							return context.dataset.borderColor;
						},
						align: "end",
						anchor: "end",
						borderRadius: 4,
						color: "white",
						font: {
							weight: "bold",
						},
						formatter: function (data, ctx) {
							if (data.highlight) {
								if (ctx.dataset.label === "price") {
									return vThis.fmtPricePerKWh(data.y, vThis.currency, true, true);
								}
								if (ctx.dataset.label === "solar") {
									return vThis.fmtWh(data.highlight, POWER_UNIT.AUTO);
								}
								return null;
							}
							return null;
						},
						padding: 6,
					},
					tooltip: {
						...commonOptions.plugins.tooltip,
						axis: "x",
						callbacks: {
							title: (tooltipItem) => {
								const { x } = tooltipItem[0].raw;
								return this.fmtFullDateTime(new Date(x));
							},
							label: () => {
								return null;
								/*
								const datasetLabel = tooltipItem.dataset.label || "";
								const value = tooltipItem.raw.y;

								// line datasets have null values
								if (tooltipItem.dataset.type === "line") {
									if (value === null) {
										return null;
									}

									const valueFmt = this.fmtW(value, POWER_UNIT.AUTO);
									return `${datasetLabel}: ${valueFmt}`;
								}

								return value
									? `${datasetLabel}: ${
											this.costType === "price" || true
												? this.fmtPricePerKWh(
														value,
														this.currency,
														true,
														true
													)
												: this.fmtGrams(value)
										}`
									: null;
								*/
							},
						},
					},
				},
				scales: {
					x: {
						type: "timeseries",
						display: true,
						time: {
							unit: "day",
						},
						ticks: {
							source: "data",
						},
						border: { display: false },
						grid: {
							display: true,
							color: colors.border,
							offset: false,
							lineWidth: function (context) {
								if (context.type !== "tick") {
									return 0;
								}
								const label = context.tick?.label;
								return Array.isArray(label) ? 1 : 0;
							},
						},
						ticks: {
							color: colors.muted,
							autoSkip: false,
							maxRotation: 0,
							minRotation: 0,
							source: "data",
							align: "center",
							callback: function (value) {
								const date = new Date(value);
								const hour = date.getHours();
								if (hour === 0) {
									return [hour, vThis.weekdayShort(date)];
								}
								if (hour % 6 === 0) {
									return hour;
								}
								return "";
							},
						},
					},
					yForecast: {
						display: false,
						border: { display: false },
						grid: { color: colors.border, drawOnChartArea: false },
						ticks: {
							callback: (value) => this.fmtW(value, POWER_UNIT.KW, true),
							color: colors.muted,
							maxTicksLimit: 3,
						},
						position: "right",
						min: 0,
					},
					yPrice: {
						display: false,
						border: { display: false },
						grid: {
							color: colors.border,
							drawOnChartArea: false,
						},
						ticks: {
							callback: (value) =>
								this.fmtPricePerKWh(value, this.currency, true, true),
							color: colors.muted,
							maxTicksLimit: 3,
						},
						position: "left",
					},
				},
			};
		},
	},
	methods: {
		filterSlots(slots: PriceSlot[] = []) {
			return slots.filter((slot) => new Date(slot.end) > this.startDate);
		},
	},
});
</script>

<style scoped>
.chart {
	width: 780px;
}

@media (min-width: 992px) {
	.chart {
		width: 100%;
	}
}
</style>
