<template>
	<div class="mb-5">
		<h3 class="fw-normal mb-3">Time Series Visualization</h3>
		<div class="chart-container my-3">
			<Chart ref="chartRef" :data="chartData" :options="chartOptions" :height="600" />
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
	BarElement,
	LineElement,
	PointElement,
	Title,
	Tooltip,
	Legend,
	type ChartOptions,
	type ChartData,
} from "chart.js";
import { Chart } from "vue-chartjs";
import type { EvoptData } from "./TimeSeriesDataTable.vue";
import type { CURRENCY } from "@/types/evcc";
import formatter from "@/mixins/formatter";
import colors, { darkenColor } from "@/colors";
import LegendList from "../Sessions/LegendList.vue";

interface ChartLegend {
	label: string;
	color: string;
	value?: string | string[];
	type?: "area" | "line";
	lineStyle?: "solid" | "dashed" | "dotted";
}

const tension = 0.1;

ChartJS.register(
	CategoryScale,
	LinearScale,
	BarElement,
	LineElement,
	PointElement,
	Title,
	Tooltip,
	Legend
);

export default defineComponent({
	name: "TimeSeriesChart",
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
		currency: {
			type: String as PropType<CURRENCY>,
			required: true,
		},
		batteryColors: {
			type: Array as PropType<string[]>,
			required: true,
		},
	},
	computed: {
		timeLabels(): string[] {
			return this.evopt.req.time_series.dt.map((_, index) => {
				const hour = index % 24;
				return `${hour}`;
			});
		},
		chartData(): ChartData {
			const datasets: any[] = [];

			// Order datasets: batteries first, then household, solar, prices
			// 1. Battery data (from response data) - power and SoC for each battery
			datasets.push(...this.getBatteryDatasets());

			// 2. Household Demand (from request data)
			datasets.push(...this.getHouseholdDatasets());

			// 3. Solar Forecast (from request data)
			datasets.push(...this.getSolarDatasets());

			// 4. Price data (from request data) - using invisible y2 axis
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
						hoverRadius: 6, // Show points on hover (same as sessions)
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
							label: (context) => {
								const label = context.dataset.label || "";
								const value = context.parsed.y;

								// Handle different axis types
								if (context.dataset.yAxisID === "y") {
									// Energy axis (kWh)
									return `${label}: ${this.formatValue(value)} kWh`;
								} else if (context.dataset.yAxisID === "y1") {
									// Power axis (kW)
									return `${label}: ${this.formatValue(value)} kW`;
								} else if (context.dataset.yAxisID === "y2") {
									// Price axis (currency/kWh)
									return `${label}: ${this.formatPrice(value)}`;
								}

								// Fallback
								return `${label}: ${this.formatValue(value)}`;
							},
						},
					},
					// Disable data labels - values only show on hover
					datalabels: {
						display: false,
					},
				},
				scales: {
					x: {
						title: {
							display: false,
						},
					},
					y: {
						type: "linear",
						position: "left",
						title: {
							display: true,
							text: "SoC (kWh)",
						},
						grid: {
							drawOnChartArea: true,
						},
						// Center zero in the middle of the chart
						afterDataLimits: (scale: any) => {
							const maxAbsValue = Math.max(Math.abs(scale.max), Math.abs(scale.min));
							scale.max = maxAbsValue;
							scale.min = -maxAbsValue;
						},
					},
					y1: {
						type: "linear",
						position: "right",
						title: {
							display: true,
							text: "Power (kW)",
						},
						grid: {
							drawOnChartArea: false,
						},
						// Center zero in the middle of the chart
						afterDataLimits: (scale: any) => {
							const maxAbsValue = Math.max(Math.abs(scale.max), Math.abs(scale.min));
							scale.max = maxAbsValue;
							scale.min = -maxAbsValue;
						},
					},
					y2: {
						type: "linear",
						position: "right",
						title: {
							display: false,
						},
						grid: {
							drawOnChartArea: false,
						},
						display: false, // Make this axis invisible
						// Center zero in the middle of the chart
						afterDataLimits: (scale: any) => {
							const maxAbsValue = Math.max(Math.abs(scale.max), Math.abs(scale.min));
							scale.max = maxAbsValue;
							scale.min = -maxAbsValue;
						},
					},
				},
			};
		},
		legends(): ChartLegend[] {
			return this.chartData.datasets
				.filter((dataset) => !dataset.hidden) // Show all datasets including battery power
				.map((dataset) => {
					const label = dataset.label || "";
					const isLine =
						label.includes("Price") ||
						label.includes("Solar Forecast") ||
						label.includes("Household Demand") ||
						label.includes("Power");

					let lineStyle: "solid" | "dashed" | "dotted" | undefined;
					if (isLine) {
						if (
							label.includes("Solar Forecast") ||
							label.includes("Household Demand") ||
							label.includes("Power")
						) {
							lineStyle = "solid";
						} else if (label.includes("Grid Import Price")) {
							lineStyle = "dashed";
						} else if (label.includes("Grid Export Price")) {
							lineStyle = "dotted";
						}
					}

					return {
						label,
						color: (dataset.backgroundColor || dataset.borderColor) as string,
						type: isLine ? "line" : "area",
						lineStyle,
					};
				});
		},
	},
	methods: {
		getSolarDatasets() {
			return [
				{
					label: "Solar Forecast",
					data: this.evopt.req.time_series.ft.map(this.convertWToKW),
					borderColor: colors.self,
					backgroundColor: colors.self,
					fill: false,
					tension,
					pointRadius: 0,
					pointHoverRadius: 6,
					borderWidth: 2,
					yAxisID: "y1",
					type: "line" as const,
				},
			];
		},
		getHouseholdDatasets() {
			// Household Demand (converted to energy for the time step)
			const householdEnergy = this.evopt.req.time_series.gt.map((power, index) => {
				const duration = this.evopt.req.time_series.dt[index] / 3600; // Convert seconds to hours
				return this.convertWhToKWh(power * duration * 1000); // Convert to Wh then kWh
			});

			// Use the next color in the palette after all battery colors
			const batteryCount = this.batteryColors.length;
			const householdColor = colors.palette[batteryCount % colors.palette.length];

			return [
				{
					label: "Household Demand",
					data: householdEnergy,
					backgroundColor: householdColor,
					borderWidth: 0,
					yAxisID: "y",
					type: "bar" as const,
					borderRadius: {
						topLeft: 10,
						topRight: 10,
					},
				},
			];
		},
		getBatteryDatasets() {
			const datasets: any[] = [];

			if (this.evopt.res.batteries?.length > 0) {
				this.evopt.res.batteries.forEach((battery, index) => {
					// Use passed battery colors instead of computing them
					const baseColor = this.batteryColors[index];
					const darkerColorValue = darkenColor(baseColor);

					// Combined charging/discharging power as one line (darker color)
					// Charging = positive, Discharging = negative
					const combinedPower = battery.charging_power.map((chargingPower, timeIndex) => {
						const dischargingPower = battery.discharging_power[timeIndex] || 0;
						const chargingKW = this.convertWToKW(chargingPower);
						const dischargingKW = this.convertWToKW(dischargingPower);

						// Return charging as positive, discharging as negative
						// One should be zero, the other should have the value
						return chargingKW > 0 ? chargingKW : -dischargingKW;
					});

					datasets.push({
						label: `Battery ${index + 1} Power`,
						data: combinedPower,
						borderColor: darkerColorValue,
						backgroundColor: darkerColorValue,
						fill: false,
						tension,
						pointRadius: 0,
						pointHoverRadius: 6,
						borderWidth: 2,
						yAxisID: "y1",
						type: "line" as const,
					});

					// SoC as bars (full color)
					datasets.push({
						label: `Battery ${index + 1} SoC`,
						data: battery.state_of_charge.map(this.convertWhToKWh),
						backgroundColor: baseColor,
						borderWidth: 0,
						yAxisID: "y",
						type: "bar" as const,
						borderRadius: {
							topLeft: 10,
							topRight: 10,
						},
					});
				});
			}

			return datasets;
		},
		getPriceDatasets() {
			const datasets: any[] = [];

			// Convert prices from raw format (€/Wh) to proper format (€/kWh) - same as table
			const convertPrice = (price: number): number => price * 1000;

			// Grid Import Price (dashed line, price color)
			datasets.push({
				label: "Grid Import Price",
				data: this.evopt.req.time_series.p_N.map(convertPrice),
				borderColor: colors.price,
				backgroundColor: colors.price,
				fill: false,
				tension,
				pointRadius: 0,
				pointHoverRadius: 6,
				borderWidth: 2,
				yAxisID: "y2", // Use invisible price axis
				type: "line" as const,
				borderDash: [10, 5], // Dashed pattern
			});

			// Grid Export Price (dotted line, price color)
			datasets.push({
				label: "Grid Export Price",
				data: this.evopt.req.time_series.p_E.map(convertPrice),
				borderColor: colors.price,
				backgroundColor: colors.price,
				fill: false,
				tension,
				pointRadius: 0,
				pointHoverRadius: 6,
				borderWidth: 2,
				yAxisID: "y2", // Use invisible price axis
				type: "line" as const,
				borderDash: [3, 6], // Dashed pattern
			});

			return datasets;
		},
		convertWToKW: (watts: number): number => {
			return watts / 1000;
		},
		convertWhToKWh: (wh: number): number => {
			return wh / 1000;
		},
		formatPrice(price: number): string {
			// Use the exact same logic as the data table: fmtPricePerKWh(value * 1000, currency, false, false)
			// But since we already multiplied by 1000 in the dataset, we use the value directly
			// and add the unit manually to match the tooltip format
			const formattedValue = this.fmtPricePerKWh(price, this.currency, false, false);
			const unit = this.pricePerKWhUnit(this.currency, false);
			return `${formattedValue} ${unit}`;
		},
		formatValue: (value: number): string => {
			return value.toFixed(2);
		},
	},
});
</script>

<style scoped>
.chart-container {
	position: relative;
	height: 600px;
	width: 100%;
}
</style>
