<template>
	<div class="mb-5">
		<div class="chart-container my-3">
			<Chart
				ref="chartRef"
				type="bar"
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
	BarElement,
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

ChartJS.register(CategoryScale, LinearScale, BarElement, Title, Tooltip, ChartLegendPlugin);

export default defineComponent({
	name: "SocChart",
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
				const currentTime = new Date(startTime.getTime() + index * 60 * 60 * 1000); // Add hours
				return currentTime.getHours().toString();
			});
		},
		chartData(): ChartData {
			const datasets: any[] = [];

			// 1. Battery SoC data
			datasets.push(...this.getBatterySocDatasets());

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
								// Percentage axis (%)
								return `${label}: ${this.formatValue(value)}%`;
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
							display: false,
						},
					},
					y: {
						type: "linear",
						position: "left",
						title: {
							display: true,
							text: "SoC (%)",
						},
						grid: {
							drawOnChartArea: true,
							color: colors.border || "",
							lineWidth: 1,
						},
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
						type: "area",
					};
				});
		},
	},
	methods: {
		getBatterySocDatasets() {
			const datasets: any[] = [];

			if (this.evopt.res.batteries?.length > 0) {
				this.evopt.res.batteries.forEach((battery, index) => {
					// Use passed battery colors
					const baseColor = this.batteryColors[index];

					// SoC as bars (full color)
					datasets.push({
						label: this.getBatteryTitle(index),
						data: battery.state_of_charge.map((socWh) =>
							this.convertWhToPercentage(socWh, index)
						),
						backgroundColor: baseColor,
						borderWidth: 0,
						yAxisID: "y",
						type: "bar" as const,
					});
				});
			}

			return datasets;
		},
		convertWhToPercentage(wh: number, batteryIndex: number): number {
			const detail = this.batteryDetails[batteryIndex];
			if (detail?.capacity && detail.capacity > 0) {
				return (wh / 1000 / detail.capacity) * 100;
			}
			return 0;
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
	height: 150px;
	width: 100%;
}
</style>
