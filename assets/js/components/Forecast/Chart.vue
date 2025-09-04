<template>
	<div>
		<div
			class="overflow-x-auto overflow-x-md-hidden chart-container border-1"
			@mouseleave="onMouseLeave"
		>
			<div style="position: relative; height: 220px" class="chart user-select-none">
				<!-- @vue-ignore -->
				<Bar ref="chart" :data="chartData" :options="options" />
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
	type ChartEvent,
	type ActiveElement,
	Chart,
} from "chart.js";
import ChartDataLabels, { type Context } from "chartjs-plugin-datalabels";
import ChartAnnotation from "chartjs-plugin-annotation";
import "chartjs-adapter-dayjs-4/dist/chartjs-adapter-dayjs-4.esm";
import { registerChartComponents, commonOptions } from "../Sessions/chartConfig";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import colors, { lighterColor, createFeedInDisabledPattern } from "@/colors";
import type { CURRENCY } from "@/types/evcc";
import { ForecastType, highestSlotIndexByDay } from "@/utils/forecast";
import type { ForecastSlot, SolarDetails, TimeseriesEntry, ForecastZone } from "./types";

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
	ChartAnnotation,
]);

export default defineComponent({
	name: "ForecastChart",
	components: { Bar },
	mixins: [formatter],
	props: {
		grid: { type: Array as PropType<ForecastSlot[]> },
		solar: { type: Object as PropType<SolarDetails> },
		co2: { type: Array as PropType<ForecastSlot[]> },
		feedin: { type: Array as PropType<ForecastSlot[]> },
		currency: { type: String as PropType<CURRENCY> },
		selected: { type: String as PropType<ForecastType> },
		feedInDisabledZones: {
			type: Array as PropType<ForecastZone[]>,
			default: () => [],
		},
	},
	emits: ["selected"],
	data(): {
		selectedIndex: number | null;
		startDate: Date;
		interval: ReturnType<typeof setTimeout> | null;
		ignoreEvents: boolean;
		ignoreEventsTimeout: ReturnType<typeof setTimeout> | null;
		animations: boolean;
	} {
		return {
			selectedIndex: null,
			startDate: new Date(),
			interval: null,
			ignoreEvents: false,
			ignoreEventsTimeout: null,
			animations: false,
		};
	},
	computed: {
		endDate() {
			const end = new Date(this.startDate);
			end.setHours(end.getHours() + 48);
			return end;
		},
		solarEntries() {
			return this.filterEntries(this.solar?.timeseries || []);
		},
		gridSlots() {
			return this.filterSlots(this.grid);
		},
		co2Slots() {
			return this.filterSlots(this.co2);
		},
		feedinSlots() {
			return this.filterSlots(this.feedin);
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
		maxFeedinIndex() {
			return this.maxIndex(this.feedinSlots);
		},
		minFeedinIndex() {
			return this.minIndex(this.feedinSlots);
		},
		maxSolarIndex() {
			return this.maxEntryIndex(this.solarEntries);
		},
		solarHighlights() {
			const { today, tomorrow, dayAfterTomorrow } = this.solar || {};
			return [
				{
					index: highestSlotIndexByDay(this.solarEntries, 0),
					energy: today?.energy,
				},
				{
					index: highestSlotIndexByDay(this.solarEntries, 1),
					energy: tomorrow?.energy,
				},
				{
					index: highestSlotIndexByDay(this.solarEntries, 2),
					energy: dayAfterTomorrow?.energy,
				},
			];
		},
		chartData() {
			const datasets = [];
			if (this.solarEntries.length > 0) {
				const active = this.selected === ForecastType.Solar;
				const color = active ? colors.self : colors.border;
				datasets.push({
					label: ForecastType.Solar,
					type: "line",
					data: this.solarEntries.map((entry, index) => {
						return {
							y: entry.val,
							x: new Date(entry.ts),
							highlight:
								active &&
								(this.selectedIndex !== null
									? this.selectedIndex === index
									: this.solarHighlights.find(({ index: i }) => i === index)
											?.energy),
						};
					}),
					yAxisID: "yForecast",
					backgroundColor: lighterColor(color),
					borderColor: color,
					fill: "origin",
					tension: 0.15,
					pointRadius: 0,
					animation: {
						y: { duration: this.animations ? 500 : 0 },
					},
					pointHoverRadius: active ? 4 : 0,
					spanGaps: true,
					order: this.getDatasetOrder(ForecastType.Solar, active),
				});
			}
			if (this.gridSlots.length > 0) {
				const active = this.selected === ForecastType.Price;
				const color = active ? colors.price : colors.border;
				datasets.push({
					label: ForecastType.Price,
					data: this.gridSlots.map((slot, index) => ({
						y: slot.value,
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
					order: this.getDatasetOrder(ForecastType.Price, active),
				});
			}
			if (this.co2Slots.length > 0) {
				const active = this.selected === ForecastType.Co2;
				const color = active ? colors.co2 : colors.border;
				datasets.push({
					label: ForecastType.Co2,
					type: "line",
					data: this.co2Slots.map((slot, index) => {
						const dataActive =
							active &&
							(this.selectedIndex !== null
								? this.selectedIndex === index
								: index === this.maxCo2Index || index === this.minCo2Index);
						return {
							y: slot.value,
							x: new Date(slot.start),
							highlight: dataActive,
							active: dataActive,
						};
					}),
					yAxisID: "yCo2",
					backgroundColor: color,
					borderColor: color,
					tension: 0.15,
					pointRadius: 0,
					pointHoverRadius: active ? 4 : 0,
					spanGaps: true,
					order: this.getDatasetOrder(ForecastType.Co2, active),
				});
			}
			if (this.feedinSlots.length > 0) {
				const active = this.selected === ForecastType.FeedIn;
				const color = active ? colors.export : colors.border;
				datasets.push({
					label: ForecastType.FeedIn,
					data: this.feedinSlots.map((slot, index) => ({
						y: slot.value,
						x: new Date(slot.start),
						highlight:
							active &&
							(this.selectedIndex !== null
								? this.selectedIndex === index
								: index === this.maxFeedinIndex || index === this.minFeedinIndex),
					})),
					yAxisID: "yPrice",
					borderRadius: 8,
					backgroundColor: color,
					borderColor: color,
					order: this.getDatasetOrder(ForecastType.FeedIn, active),
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
				events: ["mousemove", "click", "touchstart", "touchend"],
				onHover(event: ChartEvent, active: ActiveElement[], chart: Chart) {
					if (["touchend", "click"].includes(event.type)) {
						vThis.selectIndex(null, true);
						return;
					}
					const element = active.find(({ datasetIndex }) => {
						const { label } = chart.getDatasetMeta(datasetIndex);
						return label === vThis.selected;
					});
					vThis.selectIndex(element ? element.index : null);
				},
				plugins: {
					...commonOptions.plugins,
					annotation: {
						annotations: vThis.getFeedInLimitAnnotations(),
					},
					datalabels: {
						backgroundColor(context: Context) {
							return context.dataset.borderColor;
						},
						align({ chart, dataset, dataIndex }: Context) {
							const { min, max } = chart.scales["x"];
							// @ts-expect-error no-explicit-any
							const time = new Date(dataset.data[dataIndex]?.x).getTime();

							// percent along the x axis (0: start, 1: end)
							const percent = (time - min) / (max - min);
							let adjust = 0;
							const step = 20;

							// tilt label left/right if it's close to the edge
							if (percent < 0.02) {
								adjust = 2;
							} else if (percent < 0.04) {
								adjust = 1;
							} else if (percent > 0.98) {
								adjust = -2;
							} else if (percent > 0.96) {
								adjust = -1;
							}

							return -90 + adjust * step;
						},
						anchor: "end",
						offset: 8,
						padding(context: Context) {
							const data = context.dataset.data[context.dataIndex];
							// @ts-expect-error no-explicit-any
							const x = typeof data.highlight === "number" ? 32 : 8;
							return {
								x,
								y: 4,
							};
						},
						borderRadius: 4,
						color: colors.background,
						font: { weight: "bold" },
						// @ts-expect-error no-explicit-any
						formatter(value, context: Context) {
							if (value.highlight) {
								switch (context.dataset.label) {
									case ForecastType.Price:
									case ForecastType.FeedIn:
										return vThis.fmtPricePerKWh(
											value.y,
											vThis.currency,
											true,
											true
										);
									case ForecastType.Co2:
										return vThis.fmtGrams(value.y);
									case ForecastType.Solar:
										if (value.highlight === true) {
											return vThis.fmtW(value.y, POWER_UNIT.AUTO);
										} else {
											return vThis.fmtWh(value.highlight, POWER_UNIT.AUTO);
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
							// @ts-expect-error no-explicit-any
							lineWidth(context) {
								if (context.type !== "tick") {
									return 0;
								}
								const label = context.tick?.label;
								return Array.isArray(label) ? 1 : 0;
							},
						},
						min: this.startDate,
						max: this.endDate,
						ticks: {
							color: colors.muted,
							autoSkip: false,
							maxRotation: 0,
							minRotation: 0,
							source: "data",
							align: "center",
							callback(value: number) {
								const date = new Date(value);
								const hour = date.getHours();
								const minute = date.getMinutes();
								if (minute !== 0) {
									return "";
								}
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
					...(() => {
						const scales = this.calculateAlignedScales();
						return {
							yForecast: {
								...this.yScaleOptions(ForecastType.Solar),
								...scales.solar,
								beginAtZero: true,
							},
							yCo2: {
								...this.yScaleOptions(ForecastType.Co2),
								...scales.co2,
							},
							yPrice: {
								...this.yScaleOptions(ForecastType.Price, ForecastType.FeedIn),
								...scales.price,
							},
						};
					})(),
				},
			};
		},
		selectedSlot() {
			if (this.selectedIndex === null || !this.selected) return null;

			const slotMap = {
				[ForecastType.Solar]: this.solarEntries,
				[ForecastType.Price]: this.gridSlots,
				[ForecastType.Co2]: this.co2Slots,
				[ForecastType.FeedIn]: this.feedinSlots,
			};

			return slotMap[this.selected]?.[this.selectedIndex] ?? null;
		},
	},
	watch: {
		selectedSlot(slot) {
			this.$emit("selected", slot);
		},
		feedInDisabledZones() {
			this.updateChart();
		},
		selected() {
			this.updateChart();
		},
	},
	mounted() {
		this.interval = setTimeout(() => {
			this.updateStartDate();
		}, 1000 * 60);
		this.updateStartDate();
		setTimeout(() => {
			this.animations = true;
		}, 1000);
	},
	beforeUnmount() {
		if (this.interval) {
			clearTimeout(this.interval);
		}
	},
	methods: {
		updateChart() {
			// Force chart update
			this.$nextTick(() => {
				// @ts-expect-error unknown chart type
				this.$refs.chart?.chart?.update();
			});
		},
		getFeedInLimitAnnotations() {
			const annotations: Record<string, any> = {};

			// Only show annotations if zones exist
			if (!this.feedInDisabledZones || this.feedInDisabledZones.length === 0) {
				return annotations;
			}

			// Determine color based on whether feedin is selected
			const isActive = this.selected === ForecastType.FeedIn;
			const patternColor = isActive ? colors.export : colors.border;
			const patternOpacity = isActive ? 0.6 : 0.4;

			// Create annotations for each disabled zone
			this.feedInDisabledZones.forEach((zone, zoneIndex) => {
				// Bar charts center bars on their time value
				// We need to extend from half slot before start to the end
				const startTime = new Date(zone.start);
				const endTime = new Date(zone.end);

				// Calculate slot duration (typically 1 hour for feedin slots)
				const firstSlot = this.feedinSlots[0];
				const slotDuration = firstSlot
					? new Date(firstSlot.end).getTime() - new Date(firstSlot.start).getTime()
					: 3600000; // Default to 1 hour

				// Bars are centered on their time value
				// Extend start by half slot to left, reduce end by half slot to right
				const halfSlot = slotDuration / 2;
				startTime.setTime(startTime.getTime() - halfSlot);
				endTime.setTime(endTime.getTime() - halfSlot);

				annotations[`feedInDisabledZone${zoneIndex}`] = {
					type: "box",
					drawTime: "beforeDatasetsDraw",
					xMin: startTime,
					xMax: endTime,
					yMin: "min",
					yMax: "max",
					backgroundColor: (context: any) => {
						const pattern = createFeedInDisabledPattern(
							context.chart.ctx,
							patternColor || "",
							patternOpacity
						);
						if (pattern) {
							// Anchor pattern to the adjusted start of the zone
							const chart = context.chart;
							const xScale = chart.scales.x;
							const pixelX = xScale.getPixelForValue(startTime);
							pattern.setTransform(new DOMMatrix().translateSelf(pixelX, 0));
						}
						return pattern || "transparent";
					},
					borderWidth: 0,
				};
			});

			return annotations;
		},
		updateStartDate() {
			const now = new Date();
			now.setMinutes(0);
			now.setSeconds(0);
			now.setMilliseconds(0);
			this.startDate = now;
		},
		filterSlots(slots: ForecastSlot[] = []) {
			return slots.filter(
				(slot) =>
					new Date(slot.end) >= this.startDate && new Date(slot.start) <= this.endDate
			);
		},
		filterEntries(entries: TimeseriesEntry[] = []) {
			// include 1 hour before and after
			const start = new Date(this.startDate);
			start.setHours(start.getHours() - 1);
			const end = new Date(this.endDate);
			end.setHours(end.getHours() + 1);

			return entries.filter(
				(entry) => new Date(entry.ts) >= start && new Date(entry.ts) <= end
			);
		},
		onMouseLeave() {
			this.selectIndex(null, true);
		},
		selectIndex(index: number | null, timeout = false) {
			if (this.ignoreEvents) return;
			this.selectedIndex = index;

			// reset hover state (points, highlights)
			if (this.selectedIndex === null) {
				this.$nextTick(() => {
					// @ts-expect-error unknown chart type
					this.$refs.chart?.chart?.setActiveElements([]);
				});
			}

			// ignore events after selection reset because chart.js triggers delayed mousemove events
			if (timeout) {
				this.ignoreEvents = true;
				this.ignoreEventsTimeout = setTimeout(() => {
					this.ignoreEvents = false;
				}, 100);
			}
		},
		yMax(slots: ForecastSlot[] = []): number | undefined {
			const value = this.maxValue(slots);
			return value ? value * 1.15 : undefined;
		},
		yMaxEntry(entries: TimeseriesEntry[] = [], scale: number = 1): number | undefined {
			const maxValue = this.maxEntryValue(entries);
			if (!maxValue) return undefined;
			// use scale and unscaled to determine max scale
			return Math.max(maxValue * scale, maxValue) * 1.15;
		},
		maxIndex(slots: ForecastSlot[] = []) {
			return slots.reduce((max, slot, index) => {
				return slot.value > slots[max].value ? index : max;
			}, 0);
		},
		minIndex(slots: ForecastSlot[] = []) {
			return slots.reduce((min, slot, index) => {
				return slot.value < slots[min].value ? index : min;
			}, 0);
		},
		maxValue(slots: ForecastSlot[] = []) {
			return slots[this.maxIndex(slots)]?.value || null;
		},
		maxEntryValue(entries: TimeseriesEntry[] = []) {
			return entries[this.maxEntryIndex(entries)]?.val || null;
		},
		maxEntryIndex(entries: TimeseriesEntry[] = []) {
			return entries.reduce((max, entry, index) => {
				return entry.val > entries[max].val ? index : max;
			}, 0);
		},
		yScaleOptions(...types: ForecastType[]) {
			return this.selected && types.includes(this.selected)
				? {
						display: true,
						ticks: { display: false },
						border: { display: false },
						grid: {
							display: true,
							color: colors.border,
							// @ts-expect-error no-explicit-any
							lineWidth: (context) => {
								return context.tick?.value === 0 ? 1 : 0;
							},
						},
					}
				: { display: false };
		},
		getDatasetOrder(type: ForecastType, active: boolean): number {
			// Special handling for price/feedin group - they share the same Y-axis
			// and should maintain consistent order (price first, then feedin)
			const isPriceGroup = type === ForecastType.Price || type === ForecastType.FeedIn;
			const isSelectedPriceGroup =
				this.selected === ForecastType.Price || this.selected === ForecastType.FeedIn;

			if (isPriceGroup && isSelectedPriceGroup) {
				// Both price and feedin should be rendered on top when either is selected
				// Price gets order 0, feedin gets order 1 (price renders first)
				return type === ForecastType.Price ? 0 : 1;
			}

			// Standard ordering: selected type gets order 0, others get order 1
			return active ? 0 : 1;
		},
		calculateAlignedScales() {
			// Different units need individual scales but aligned 0 lines
			// Challenge: Price/feedin can be negative, solar/CO₂ are always positive
			// Solution: Adjust all scales so 0 appears at the same relative position

			const solarMax = this.yMaxEntry(this.solarEntries, this.solar?.scale) || 10;
			const co2Max = this.yMax(this.co2Slots) || 500;

			// For price data, we need to find both min and max values
			const priceValues = [
				...(this.gridSlots?.map((slot) => slot.value) || []),
				...(this.feedinSlots?.map((slot) => slot.value) || []),
			].filter((val) => val != null);

			let priceMin = 0;
			let priceMax = 0.5;

			if (priceValues.length > 0) {
				const actualMin = Math.min(...priceValues);
				const actualMax = Math.max(...priceValues);

				// Add 15% padding like original yMax method
				priceMin = actualMin < 0 ? actualMin * 1.15 : 0;
				priceMax = actualMax > 0 ? actualMax * 1.15 : 0;
			}

			// Calculate where 0 should be positioned (as ratio from bottom)
			// If price has negative values, 0 is not at the bottom
			const priceRange = priceMax - priceMin;
			const zeroPosition = priceRange > 0 ? Math.abs(priceMin) / priceRange : 0;

			// Adjust solar and co2 scales to position their 0 at the same relative position
			let adjustedSolarMin = 0;
			let adjustedCo2Min = 0;

			if (zeroPosition > 0 && zeroPosition < 1) {
				// 0 is not at bottom, need to add negative space to solar/co2
				adjustedSolarMin = -solarMax * (zeroPosition / (1 - zeroPosition));
				adjustedCo2Min = -co2Max * (zeroPosition / (1 - zeroPosition));
			}

			return {
				solar: {
					min: adjustedSolarMin,
					max: solarMax,
				},
				co2: {
					min: adjustedCo2Min,
					max: co2Max,
				},
				price: {
					min: priceMin,
					max: priceMax,
				},
			};
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
