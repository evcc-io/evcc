<template>
	<div>
		<div style="position: relative; height: 300px" class="my-3">
			<Bar :data="chartData" :options="options" />
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
import "chartjs-adapter-dayjs-4/dist/chartjs-adapter-dayjs-4.esm";
import { registerChartComponents, commonOptions } from "./Sessions/chartConfig";
import formatter, { POWER_UNIT } from "../mixins/formatter";
import colors, { lighterColor } from "../colors";
import type { PriceSlot } from "../utils/forecast";
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
		chartData() {
			const datasets = [];
			if (this.solarSlots.length > 0) {
				datasets.push({
					label: "solar forecast",
					type: "line",
					data: this.solarSlots.map((slot) => ({
						y: slot.price,
						x: new Date(slot.start),
					})),
					yAxisID: "yForecast",
					backgroundColor: lighterColor(colors.self),
					borderColor: colors.self,
					fill: "origin",
					tension: 0.25,
					pointRadius: 0,
					pointHoverRadius: 6,
					borderWidth: 2,
					spanGaps: true,
				});
			}
			if (this.gridSlots.length > 0) {
				datasets.push({
					label: "grid price",
					data: this.gridSlots.map((slot) => ({
						y: slot.price,
						x: new Date(slot.start),
					})),
					yAxisID: "yPrice",
					borderRadius: 2,
					backgroundColor: colors.light,
					borderColor: colors.light,
				});
			}
			return {
				labels: Array.from(
					{ length: 48 },
					(_, i) => new Date(this.startDate.getTime() + i * 60 * 60 * 1000)
				),
				datasets,
			};
		},
		options() {
			const vThis = this;
			return {
				...commonOptions,
				locale: this.$i18n?.locale,
				color: colors.text,
				borderSkipped: false,
				maxBarThickness: 40,
				animation: false,
				interaction: {
					mode: "nearest",
				},
				plugins: {
					...commonOptions.plugins,
					tooltip: {
						...commonOptions.plugins.tooltip,
						axis: "x",
						callbacks: {
							title: (tooltipItem) => {
								const { x } = tooltipItem[0].raw;
								return this.fmtFullDateTime(new Date(x));
							},
							label: (tooltipItem) => {
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
							align: "center",
							callback: function (value) {
								const date = new Date(value);
								const hour = date.getHours();
								const mins = date.getMinutes();
								console.log(date, hour, mins);
								if (mins !== 0) {
									return "";
								}
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
						border: { display: false },
						grid: { color: colors.border },
						title: {
							text: "kW",
							display: true,
							color: colors.muted,
						},
						ticks: {
							callback: (value) => this.fmtW(value, POWER_UNIT.KW, false),
							color: colors.muted,
							maxTicksLimit: 6,
						},
						position: "right",
						min: 0,
					},
					yPrice: {
						title: {
							text: this.pricePerKWhUnit(this.currency),
							display: true,
							color: colors.muted,
						},
						border: { display: false },
						grid: {
							color: colors.border,
							drawOnChartArea: false,
						},
						ticks: {
							callback: (value) =>
								this.fmtPricePerKWh(value, this.currency, true, false),
							color: colors.muted,
							maxTicksLimit: 6,
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
