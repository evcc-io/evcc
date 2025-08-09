<template>
	<div class="mb-5">
		<h4 class="fw-normal mb-3">Battery SoC</h4>
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
import type { CURRENCY } from "@/types/evcc";
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
								// Energy axis (kWh)
								return `${label}: ${this.formatValue(value)} kWh`;
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
						min: 0, // Set minimum to zero as requested
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
		convertWhToKWh: (wh: number): number => {
			return wh / 1000;
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
	height: 150px;
	width: 100%;
}
</style>
