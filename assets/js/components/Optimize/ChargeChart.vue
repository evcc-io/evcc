<template>
	<div class="mb-5">
		<div class="chart-container my-3">
			<Chart
				ref="chartRef"
				type="bar"
				:data="chartData"
				:options="chartOptions"
				:height="300"
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
	BarController,
	BarElement,
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
import type { CURRENCY, BatteryDetail } from "@/types/evcc";
import formatter from "@/mixins/formatter";
import colors from "@/colors";
import LegendList from "../Sessions/LegendList.vue";
import type { Legend } from "../Sessions/types";

ChartJS.register(
	CategoryScale,
	LinearScale,
	BarController,
	BarElement,
	LineElement,
	PointElement,
	Title,
	Tooltip,
	ChartLegendPlugin
);

export default defineComponent({
	name: "ChargeChart",
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
		batteryDetails: {
			type: Array as PropType<BatteryDetail[]>,
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
		batteryColors: {
			type: Array as PropType<string[]>,
			default: () => [],
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

			// 1. Grid power data (import/export)
			datasets.push(...this.getGridPowerDatasets());

			// 2. Solar Forecast
			datasets.push(...this.getSolarDatasets());

			// 3. Household Demand (power)
			datasets.push(...this.getHouseholdDatasets());

			// 4. Battery power data
			datasets.push(...this.getBatteryPowerDatasets());

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
				hover: {
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
								// Special handling for Grid Power
								if (label === "Grid Power") {
									if (value > 0) {
										return `Grid Import: ${this.formatValue(Math.abs(value))} kW`;
									} else if (value < 0) {
										return `Grid Export: ${this.formatValue(Math.abs(value))} kW`;
									} else {
										return `Grid: 0 kW`;
									}
								}
								// Power axis (kW)
								return `${label}: ${this.formatValue(value)} kW`;
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
						stacked: true,
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
							text: "Power (kW)",
						},
						stacked: true,
						grid: {
							drawOnChartArea: true,
							color: colors.border || "",
							lineWidth: 1,
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
					const isLine = dataset.type === "line";

					return {
						label,
						color: (dataset.backgroundColor || dataset.borderColor) as string,
						value: "", // Required by Legend type, but not used in this context
						type: isLine ? "line" : "area",
					};
				});
		},
	},
	methods: {
		getSolarDatasets() {
			return [
				{
					label: "Solar Forecast",
					data: this.evopt.req.time_series.ft.map(this.convertWhToKW),
					borderColor: colors.self,
					backgroundColor: colors.self,
					fill: false,
					tension: 0.2,
					borderJoinStyle: "round",
					borderCapStyle: "round",
					pointRadius: 0,
					pointHoverRadius: 6,
					borderWidth: 3,
					yAxisID: "y",
					type: "line" as const,
					stack: "solar",
				},
			];
		},
		getBatteryPowerDatasets() {
			const datasets: any[] = [];

			if (this.evopt.res.batteries?.length > 0) {
				this.evopt.res.batteries.forEach((battery, index) => {
					// Use passed battery colors (same as SoC)
					const baseColor = this.batteryColors[index];

					// Combined charging/discharging power as one line (same color as SoC)
					// Charging = positive, Discharging = negative
					const combinedPower = battery.charging_power.map((chargingPower, timeIndex) => {
						const dischargingPower = battery.discharging_power[timeIndex] || 0;
						const chargingKW = this.convertWhToKW(chargingPower, timeIndex);
						const dischargingKW = this.convertWhToKW(dischargingPower, timeIndex);

						// Return charging as positive, discharging as negative
						// One should be zero, the other should have the value
						return chargingKW > 0 ? chargingKW : -dischargingKW;
					});

					datasets.push({
						label: this.getBatteryTitle(index),
						data: combinedPower,
						backgroundColor: baseColor,
						borderWidth: 0,
						yAxisID: "y",
						type: "bar" as const,
						stack: "charge",
					});
				});
			}

			return datasets;
		},
		getHouseholdDatasets() {
			const householdPower = this.evopt.req.time_series.gt.map(this.convertWhToKW);

			// Use the next color in the palette after all battery colors
			const batteryCount = this.batteryColors.length;
			const householdColor = colors.palette[batteryCount % colors.palette.length];

			return [
				{
					label: "Household",
					data: householdPower,
					backgroundColor: householdColor,
					borderWidth: 0,
					yAxisID: "y",
					type: "bar" as const,
					stack: "charge",
				},
			];
		},

		getGridPowerDatasets() {
			const datasets: any[] = [];

			// Get grid import and export data
			const gridImport = this.evopt.res.grid_import || [];
			const gridExport = this.evopt.res.grid_export || [];

			// Combine grid import and export into a single line
			// Grid import is positive, grid export is negative (one is always zero)
			const gridPower = gridImport.map((importValue, index) => {
				const exportValue = gridExport[index] || 0;
				const importKW = this.convertWhToKW(importValue, index);
				const exportKW = this.convertWhToKW(exportValue, index);
				// Return import as positive, export as negative
				return importKW > 0 ? importKW : -exportKW;
			});

			datasets.push({
				label: "Grid Power",
				data: gridPower,
				borderColor: "#666666", // Dark gray
				backgroundColor: "#666666", // Dark gray
				fill: false,
				tension: 0.2,
				borderWidth: 2, // Same thickness as price chart lines
				pointRadius: 0,
				pointHoverRadius: 6,
				yAxisID: "y",
				type: "line" as const,
				stack: "grid",
			});

			return datasets;
		},

		convertWhToKW(wh: number, index: number): number {
			// Convert Wh to kW by normalizing against time duration
			// Power (kW) = Energy (Wh) / Time (h) / 1000
			const dtSeconds = this.evopt.req.time_series.dt[index] || 0;
			const hours = dtSeconds / 3600; // Convert seconds to hours
			return wh / hours / 1000;
		},

		formatValue: (value: number): string => {
			return value.toFixed(2);
		},

		getBatteryTitle(index: number): string {
			const detail = this.batteryDetails[index];
			return detail ? detail.title || detail.name : `Battery ${index + 1}`;
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
	height: 300px;
	width: 100%;
}
</style>
