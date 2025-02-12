<template>
	<div>
		<div
			class="overflow-x-auto overflow-x-md-auto chart-container border-1"
			@mouseleave="onMouseLeave"
		>
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
import {
	energyByDay,
	highestSlotIndexByDay,
	type PriceSlot,
	ForecastType,
} from "../utils/forecast";

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
		selected: { type: String as PropType<ForecastType> },
	},
	emits: ["selected"],
	data() {
		return {
			selectedIndex: null,
		};
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
		co2Slots() {
			return this.filterSlots(this.co2);
		},
		currentSlots() {
			switch (this.selected) {
				case ForecastType.Price:
					return this.gridSlots;
				case ForecastType.Solar:
					return this.solarSlots;
				case ForecastType.Co2:
					return this.co2Slots;
				default:
					return [];
			}
		},
		selectedSlot() {
			return this.selectedIndex !== null ? this.currentSlots[this.selectedIndex] : null;
		},
		maxPriceIndex() {
			return this.maxIndex(this.gridSlots);
		},
		minPriceIndex() {
			return this.minIndex(this.gridSlots);
		},
		maxCo2Index() {
			return this.maxIndex(this.co2Slots);
		},
		minCo2Index() {
			return this.minIndex(this.co2Slots);
		},
		maxSolarIndex() {
			return this.maxIndex(this.solarSlots);
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
				const active = this.selected === ForecastType.Solar;
				const color = active ? colors.self : colors.border;
				datasets.push({
					label: ForecastType.Solar,
					type: "line",
					data: this.solarSlots.map((slot, index) => ({
						y: slot.price,
						x: new Date(slot.start),
						highlight:
							active &&
							(this.selectedIndex !== null
								? this.selectedIndex === index
								: this.solarHighlights.find(({ index: i }) => i === index)?.energy),
					})),
					yAxisID: "yForecast",
					backgroundColor: lighterColor(color),
					borderColor: color,
					fill: "origin",
					tension: 0.5,
					pointRadius: 0,
					pointHoverRadius: active ? 4 : 0,
					spanGaps: true,
					order: active ? 0 : 1,
				});
			}
			if (this.gridSlots.length > 0) {
				const active = this.selected === ForecastType.Price;
				const color = active ? colors.price : colors.border;
				datasets.push({
					label: ForecastType.Price,
					data: this.gridSlots.map((slot, index) => ({
						y: slot.price,
						x: new Date(slot.start),
						highlight:
							active &&
							(this.selectedIndex !== null
								? this.selectedIndex === index
								: index === this.maxPriceIndex || index === this.minPriceIndex),
					})),
					yAxisID: "yPrice",
					borderRadius: 8,
					backgroundColor: color,
					borderColor: color,
					order: active ? 0 : 1,
				});
			}
			if (this.co2Slots.length > 0) {
				const active = this.selected === ForecastType.Co2;
				const color = active ? colors.co2 : colors.border;
				datasets.push({
					label: ForecastType.Co2,
					type: "line",
					data: this.co2Slots.map((slot, index) => ({
						y: slot.price,
						x: new Date(slot.start),
						highlight:
							active &&
							(this.selectedIndex !== null
								? this.selectedIndex === index
								: index === this.maxCo2Index || index === this.minCo2Index),
					})),
					yAxisID: "yCo2",
					backgroundColor: color,
					borderColor: color,
					tension: 0.25,
					pointRadius: 0,
					pointHoverRadius: active ? 4 : 0,
					spanGaps: true,
					order: active ? 0 : 1,
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
				animation: {
					duration: 500, // --evcc-transition-medium
					colors: true,
					numbers: false,
				},
				interaction: {
					mode: "index",
					axis: "x",
					intersect: false,
				},
				categoryPercentage: 0.7,
				onHover: function (event, active) {
					const element = active.find(({ datasetIndex }) => {
						const { label } = event.chart.getDatasetMeta(datasetIndex);
						return label === vThis.selected;
					});
					vThis.selectedIndex = element ? element.index : null;
				},
				plugins: {
					...commonOptions.plugins,
					datalabels: {
						backgroundColor: function (context) {
							return context.dataset.borderColor;
						},
						align: function (context) {
							// rotate label position to avoid horizontal clipping
							const step = 20;
							const adjust = {
								0: step * 2,
								1: step * 1,
								46: step * -1,
								47: step * -2,
							};
							return -90 + (adjust[context.dataIndex] || 0);
						},
						anchor: "end",
						offset: 8,
						borderRadius: 4,
						color: colors.background,
						font: { weight: "bold" },
						formatter: function (data, ctx) {
							if (data.highlight) {
								switch (ctx.dataset.label) {
									case ForecastType.Price:
										return vThis.fmtPricePerKWh(
											data.y,
											vThis.currency,
											true,
											true
										);
									case ForecastType.Co2:
										return vThis.fmtGrams(data.y);
									case ForecastType.Solar:
										if (data.highlight === true) {
											return vThis.fmtW(data.y, POWER_UNIT.AUTO);
										} else {
											return vThis.fmtWh(data.highlight, POWER_UNIT.AUTO);
										}
									default:
										return null;
								}
							}
							return null;
						},
					},
					tooltip: null,
				},
				scales: {
					x: {
						type: "timeseries",
						display: true,
						time: { unit: "day" },
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
								const hourFmt = vThis.hourShort(date);
								if (hour === 0) {
									return [hourFmt, vThis.weekdayShort(date)];
								}
								if (hour % 6 === 0) {
									return hourFmt;
								}
								return "";
							},
						},
					},
					yForecast: { display: false, min: 0, max: this.yMax(this.solarSlots) },
					yCo2: { display: false, min: 0, max: this.yMax(this.co2Slots) },
					yPrice: { display: false, min: 0, max: this.yMax(this.gridSlots) },
				},
			};
		},
	},
	watch: {
		selectedSlot(slot) {
			this.$emit("selected", slot);
		},
	},
	methods: {
		filterSlots(slots: PriceSlot[] = []) {
			return slots.filter((slot) => new Date(slot.end) > this.startDate).slice(0, 48);
		},
		onMouseLeave() {
			this.selectedIndex = null;
		},
		yMax(slots: PriceSlot[] = []): number | undefined {
			const value = this.maxValue(slots);
			return value ? value * 1.15 : undefined;
		},
		maxIndex(slots: PriceSlot[] = []) {
			return slots.reduce((max, slot, index) => {
				return slot.price > slots[max].price ? index : max;
			}, 0);
		},
		minIndex(slots: PriceSlot[] = []) {
			return slots.reduce((min, slot, index) => {
				return slot.price < slots[min].price ? index : min;
			}, 0);
		},
		maxValue(slots: PriceSlot[] = []) {
			return slots[this.maxIndex(slots)]?.price || null;
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
