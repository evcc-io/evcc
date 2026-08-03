<template>
	<div>
		<div class="chart-container my-3" @mouseleave="emitHoverIndex(null)">
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
	LineController,
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
import { syncChartTooltip } from "./chartSync";
import { robustPriceMax, PRICE_SPIKE_CLIP } from "@/utils/robustPriceMax";

const tension = 0;

ChartJS.register(
	CategoryScale,
	LinearScale,
	LineController,
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
		gridForecastMissing: {
			type: Array as PropType<boolean[]>,
			default: () => [],
		},
		currency: {
			type: String as PropType<CURRENCY>,
			required: true,
		},
		activeIndex: {
			type: Number as PropType<number | null>,
			default: null,
		},
	},
	emits: ["hover-index"],
	computed: {
		priceAxisMax(): number | undefined {
			const ts = this.evopt?.req?.time_series;
			if (!ts) return undefined;
			const missing = this.gridForecastMissing || [];
			const factor = this.pricePerKWhDisplayFactor(this.currency);
			const convert = (p: number) => p * 1000 * factor;
			const values = [
				...(ts.p_N || []).filter((_, i) => !missing[i]).map(convert),
				...(ts.p_E || []).map(convert),
			];
			// values are in display units (e.g. cents), so scale the base threshold
			return values.length
				? robustPriceMax(values, { threshold: PRICE_SPIKE_CLIP * factor })
				: undefined;
		},
		// resolved danger colour used to flag spikes clipped at the axis ceiling
		spikeColor(): string {
			if (typeof getComputedStyle === "undefined") return "#dc3545";
			return (
				getComputedStyle(document.documentElement).getPropertyValue("--bs-danger").trim() ||
				"#dc3545"
			);
		},
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
				onHover: (_event, activeElements) => {
					this.emitHoverIndex(activeElements[0]?.index ?? null);
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
									const step = window.innerWidth < 576 ? 6 : 4;
									if (hour % step === 0) {
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
							text: this.pricePerKWhUnit(this.currency, false),
						},
						grid: {
							drawOnChartArea: true,
						},
						// cap the top at a robust percentile so rare price spikes don't
						// flatten the everyday range (tooltip still shows the real value)
						max: this.priceAxisMax,
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
	watch: {
		activeIndex() {
			this.syncTooltip();
		},
	},
	methods: {
		getChart() {
			return (this.$refs["chartRef"] as { chart?: ChartJS } | undefined)?.chart;
		},
		emitHoverIndex(index: number | null) {
			this.$emit("hover-index", index);
		},
		syncTooltip() {
			syncChartTooltip(this.getChart(), this.activeIndex);
		},
		getPriceDatasets() {
			const datasets: any[] = [];

			// Convert raw price (currency/Wh) to the display unit per kWh (e.g. ct/kWh)
			const factor = this.pricePerKWhDisplayFactor(this.currency);
			const convertPrice = (price: number): number => price * 1000 * factor;

			const cap = this.priceAxisMax;
			// a point sits above the axis cap -> it's a clipped spike; mark it
			const clipped = (v: number | null) => v != null && cap != null && v > cap;

			// Grid Import Price (solid line, price color)
			// Slots without a real planner tariff are filled with a fallback rate on
			// the backend; gap those points so the fallback value is not drawn.
			const importData = this.evopt.req.time_series.p_N.map((price, index) =>
				this.gridForecastMissing[index] ? null : convertPrice(price)
			);
			datasets.push({
				label: "Import",
				data: importData,
				borderColor: colors.grid,
				backgroundColor: colors.grid,
				fill: false,
				tension,
				stepped: true,
				borderJoinStyle: "round",
				borderCapStyle: "round",
				pointRadius: importData.map((v) => (clipped(v) ? 3.5 : 0)),
				pointBackgroundColor: importData.map((v) =>
					clipped(v) ? this.spikeColor : colors.grid
				),
				pointBorderColor: importData.map((v) =>
					clipped(v) ? this.spikeColor : colors.grid
				),
				pointHoverRadius: 6,
				borderWidth: 2,
				yAxisID: "y",
				type: "line" as const,
			});

			// Grid Export Price (solid line, price color)
			const exportData = this.evopt.req.time_series.p_E.map(convertPrice);
			datasets.push({
				label: "Export",
				data: exportData,
				borderColor: colors.price,
				backgroundColor: colors.price,
				pointRadius: exportData.map((v) => (clipped(v) ? 3.5 : 0)),
				pointBackgroundColor: exportData.map((v) =>
					clipped(v) ? this.spikeColor : colors.price
				),
				pointBorderColor: exportData.map((v) =>
					clipped(v) ? this.spikeColor : colors.price
				),
				fill: false,
				tension,
				stepped: true,
				borderJoinStyle: "round",
				borderCapStyle: "round",
				pointHoverRadius: 6,
				borderWidth: 2,
				yAxisID: "y",
				type: "line" as const,
			});

			return datasets;
		},
		formatPrice(price: number): string {
			// price is already in display unit (see convertPrice); undo the factor so the
			// formatter re-applies it and appends the matching unit
			const factor = this.pricePerKWhDisplayFactor(this.currency);
			return this.fmtPricePerKWh(price / factor, this.currency);
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
