<template>
	<div class="mb-5">
		<div class="chart-container my-3">
			<Chart
				ref="chartRef"
				type="line"
				:data="chartData"
				:options="chartOptions"
				:height="150"
			/>
		</div>
		<LegendList :legends="legends" />
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import {
	Chart as ChartJS,
	CategoryScale,
	LinearScale,
	LineElement,
	PointElement,
	Title,
	Tooltip,
	Legend as ChartLegendPlugin,
	type ChartOptions,
	type ChartData,
} from "chart.js";
import { Chart } from "vue-chartjs";
import type { EvoptData } from "./TimeSeriesDataTable.vue";
import type { CURRENCY } from "@/types/evcc";
import formatter from "@/mixins/formatter";
import colors from "@/colors";
import LegendList from "../Sessions/LegendList.vue";
import type { Legend } from "../Sessions/types";

const tension = 0;

ChartJS.register(
	CategoryScale,
	LinearScale,
	LineElement,
	PointElement,
	Title,
	Tooltip,
	ChartLegendPlugin
);

export default defineComponent({
	name: "PriceChart",
	components: {
		Chart,
		LegendList,
	},
	mixins: [formatter],
	props: {
		evopt: {
			type: Object as PropType<EvoptData>,
			required: true,
		},
		timestamp: {
			type: String,
			default: "",
		},
		currency: {
			type: String as PropType<CURRENCY>,
			required: true,
		},
	},
	computed: {
		timeLabels(): string[] {
			const startTime = new Date(this.timestamp);
			return this.evopt.req.time_series.dt.map((_, index) => {
				// Calculate cumulative time from dt array
				let cumulativeSeconds = 0;
				for (let i = 0; i < index; i++) {
					cumulativeSeconds += this.evopt.req.time_series.dt[i] || 0;
				}

				const currentTime = new Date(startTime.getTime() + cumulativeSeconds * 1000);
				const hour = currentTime.getHours();
				const minute = currentTime.getMinutes();

				// Only show labels at exact hour boundaries divisible by 4
				if (minute === 0 && hour % 4 === 0) {
					return hour.toString();
				}
				return "";
			});
		},
		chartData(): ChartData {
			const datasets: any[] = [];

			// Price data only
			datasets.push(...this.getPriceDatasets());

			return {
				labels: this.timeLabels,
				datasets: datasets,
			};
		},
		chartOptions(): ChartOptions {
			return {
				responsive: true,
				maintainAspectRatio: false,
				color: colors.text || "",
				animation: false,
				interaction: {
					mode: "index",
					intersect: false,
				},
				elements: {
					point: {
						radius: 0, // Hide points by default
						hoverRadius: 6, // Show points on hover
					},
				},
				plugins: {
					title: { display: false },
					legend: { display: false },
					tooltip: {
						backgroundColor: "#000000cc",
						boxPadding: 5,
						usePointStyle: false,
						borderWidth: 0.00001,
						mode: "index",
						intersect: false,
						callbacks: {
							title: (context) => {
								const index = context[0]?.dataIndex;
								return this.formatTimeRange(index ?? 0);
							},
							label: (context) => {
								const label = context.dataset.label || "";
								const value = context.parsed.y ?? 0;
								// Price axis (currency/kWh)
								return `${label}: ${this.formatPrice(value)}`;
							},
						},
					},
					datalabels: {
						display: false,
					},
				},
				scales: {
					x: {
						title: {
							display: false,
						},
						grid: {
							display: true,
							drawOnChartArea: true,
							drawTicks: true,
							color: "transparent",
							tickLength: 4,
						},
						ticks: {
							autoSkip: false,
							maxRotation: 0,
							minRotation: 0,
							callback: (_value, index) => {
								const startTime = new Date(this.timestamp);

								// Calculate cumulative time from dt array
								let cumulativeSeconds = 0;
								for (let i = 0; i < index; i++) {
									cumulativeSeconds += this.evopt.req.time_series.dt[i] || 0;
								}

								const currentTime = new Date(
									startTime.getTime() + cumulativeSeconds * 1000
								);
								const hour = currentTime.getHours();
								const minute = currentTime.getMinutes();

								// Show ticks at exact hour boundaries
								if (minute === 0) {
									// Show labels only at hours divisible by 4
									if (hour % 4 === 0) {
										return hour.toString();
									}
									// Show tick but no label for other hours
									return "";
								}
								// Return undefined to skip this tick entirely
								return undefined;
							},
						},
					},
					y: {
						type: "linear",
						position: "left",
						title: {
							display: true,
							text: `Price (${this.pricePerKWhUnit(this.currency, false)})`,
						},
						grid: {
							drawOnChartArea: true,
						},
						// Keep scales purely based on values, no fixed boundaries
					},
				},
			};
		},
		legends(): Legend[] {
			return this.chartData.datasets
				.filter((dataset) => !dataset.hidden)
				.map((dataset) => {
					const label = dataset.label || "";

					return {
						label,
						color: (dataset.backgroundColor || dataset.borderColor) as string,
						value: "", // Required by Legend type, but not used in this context
						type: "line",
					};
				});
		},
	},
	methods: {
		getPriceDatasets() {
			const datasets: any[] = [];

			// Convert prices from raw format (€/Wh) to proper format (€/kWh) - same as table
			const convertPrice = (price: number): number => price * 1000;

			// Grid Import Price (solid line, price color)
			datasets.push({
				label: "Import",
				data: this.evopt.req.time_series.p_N.map(convertPrice),
				borderColor: colors.grid,
				backgroundColor: colors.grid,
				fill: false,
				tension,
				stepped: true,
				borderJoinStyle: "round",
				borderCapStyle: "round",
				pointRadius: 0,
				pointHoverRadius: 6,
				borderWidth: 2,
				yAxisID: "y",
				type: "line" as const,
			});

			// Grid Export Price (solid line, price color)
			datasets.push({
				label: "Export",
				data: this.evopt.req.time_series.p_E.map(convertPrice),
				borderColor: colors.price,
				backgroundColor: colors.price,
				fill: false,
				tension,
				stepped: true,
				borderJoinStyle: "round",
				borderCapStyle: "round",
				pointRadius: 0,
				pointHoverRadius: 6,
				borderWidth: 2,
				yAxisID: "y",
				type: "line" as const,
			});

			return datasets;
		},
		formatPrice(price: number): string {
			// Use the exact same logic as the data table: fmtPricePerKWh(value * 1000, currency, false, false)
			// But since we already multiplied by 1000 in the dataset, we use the value directly
			// and add the unit manually to match the tooltip format
			const formattedValue = this.fmtPricePerKWh(price, this.currency, false, false);
			const unit = this.pricePerKWhUnit(this.currency, false);
			return `${formattedValue} ${unit}`;
		},

		formatTimeRange(index: number): string {
			const startTime = new Date(this.timestamp);

			// Calculate cumulative time from dt array
			let cumulativeSeconds = 0;
			for (let i = 0; i < index; i++) {
				cumulativeSeconds += this.evopt.req.time_series.dt[i] || 0;
			}

			const slotStart = new Date(startTime.getTime() + cumulativeSeconds * 1000);
			const slotDuration = this.evopt.req.time_series.dt[index] || 0;
			const slotEnd = new Date(slotStart.getTime() + slotDuration * 1000);

			const formatTime = (date: Date): string => {
				const hours = date.getHours().toString().padStart(2, "0");
				const minutes = date.getMinutes().toString().padStart(2, "0");
				return `${hours}:${minutes}`;
			};

			return `${formatTime(slotStart)} - ${formatTime(slotEnd)}`;
		},
	},
});
</script>

<style scoped>
.chart-container {
	position: relative;
	height: 150px;
	width: 100%;
}
</style>
