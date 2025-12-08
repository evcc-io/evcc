<template>
	<div class="mb-5">
		<div v-for="(_battery, index) in evopt.res.batteries" :key="index" class="mb-3">
			<div class="mb-2" style="font-size: 0.875rem; font-weight: bold">
				{{ getBatteryTitle(index) }}
			</div>
			<div
				:class="
					index === evopt.res.batteries.length - 1
						? 'chart-container-small-with-labels'
						: 'chart-container-small'
				"
			>
				<Chart
					:ref="`chartRef${index}`"
					type="bar"
					:data="getBatteryChartData(index)"
					:options="getBatteryChartOptions(index)"
					:height="75"
				/>
			</div>
		</div>
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

ChartJS.register(CategoryScale, LinearScale, BarElement, Title, Tooltip, ChartLegendPlugin);

export default defineComponent({
	name: "SocChart",
	components: {
		Chart,
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
	computed: {},
	methods: {
		getTimeLabels(): string[] {
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

		getBatteryChartData(batteryIndex: number): ChartData {
			const battery = this.evopt.res.batteries[batteryIndex];
			if (!battery) {
				return { labels: [], datasets: [] };
			}
			const baseColor = this.batteryColors[batteryIndex] || "";

			return {
				labels: this.getTimeLabels(),
				datasets: [
					{
						label: this.getBatteryTitle(batteryIndex),
						data: battery.state_of_charge.map((socWh) =>
							this.convertWhToPercentage(socWh, batteryIndex)
						),
						backgroundColor: baseColor,
						borderWidth: 0,
						yAxisID: "y",
					},
				],
			};
		},

		getBatteryChartOptions(batteryIndex: number): ChartOptions {
			const isLastBattery = batteryIndex === this.evopt.res.batteries.length - 1;
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
							title: (context) => {
								const index = context[0]?.dataIndex;
								return this.formatTimeRange(index ?? 0);
							},
							label: (context) => {
								const value = context.parsed.y ?? 0;
								return `SoC: ${this.formatValue(value)}%`;
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
							display: isLastBattery,
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
						min: 0,
						max: 100,
						title: {
							display: false,
						},
						grid: {
							drawOnChartArea: true,
							color: colors.border || "",
							lineWidth: 1,
						},
						ticks: {
							callback: (value) => `${value}%`,
						},
					},
				},
			};
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
.chart-container-small {
	position: relative;
	height: 75px;
	width: 100%;
}

.chart-container-small-with-labels {
	position: relative;
	height: 95px;
	width: 100%;
}

.battery-indicator {
	width: 1rem;
	height: 1rem;
	border-radius: 50%;
	flex-shrink: 0;
}
</style>
