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
			const startTime = new Date(this.timestamp);
			return this.evopt.req.time_series.dt.map((_, index) => {
				const currentTime = new Date(startTime.getTime() + index * 60 * 60 * 1000); // Add hours
				return currentTime.getHours().toString();
			});
		},
		chartData(): ChartData {
			const datasets: any[] = [];

			// 1. Solar Forecast (first, with increased tension)
			datasets.push(...this.getSolarDatasets());

			// 2. Battery power data
			datasets.push(...this.getBatteryPowerDatasets());

			// 3. Household Demand (power)
			datasets.push(...this.getHouseholdDatasets());

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
							label: (context) => {
								const label = context.dataset.label || "";
								const value = context.parsed.y;
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
							display: false,
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
							color: (context: any) => {
								// Make zero axis black to highlight
								if (context.tick?.value === 0) {
									return "#000000";
								}
								return colors.border || "#e0e0e0";
							},
							lineWidth: (context: any) => {
								// Make zero axis slightly thicker
								if (context.tick?.value === 0) {
									return 2;
								}
								return 1;
							},
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
					data: this.evopt.req.time_series.ft.map(this.convertWToKW),
					borderColor: colors.self,
					backgroundColor: colors.self,
					fill: false,
					tension: 0.25,
					borderJoinStyle: "round",
					borderCapStyle: "round",
					pointRadius: 0,
					pointHoverRadius: 6,
					borderWidth: 3,
					yAxisID: "y",
					type: "line" as const,
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
						const chargingKW = this.convertWToKW(chargingPower);
						const dischargingKW = this.convertWToKW(dischargingPower);

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
						stack: "power",
					});
				});
			}

			return datasets;
		},
		getHouseholdDatasets() {
			// Household Demand (as power, not energy like in SocChart)
			const householdPower = this.evopt.req.time_series.gt.map(this.convertWToKW);

			// Use the next color in the palette after all battery colors
			const batteryCount = this.batteryColors.length;
			const householdColor = colors.palette[batteryCount % colors.palette.length];

			return [
				{
					label: "Household Demand",
					data: householdPower,
					backgroundColor: householdColor,
					borderWidth: 0,
					yAxisID: "y",
					type: "bar" as const,
					stack: "power",
				},
			];
		},

		convertWToKW: (watts: number): number => {
			return watts / 1000;
		},

		formatValue: (value: number): string => {
			return value.toFixed(2);
		},

		getBatteryTitle(index: number): string {
			const detail = this.batteryDetails[index];
			return detail ? detail.title || detail.name : `Battery ${index + 1}`;
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
