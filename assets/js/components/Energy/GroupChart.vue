<template>
	<div class="history-chart-wrapper" :data-testid="`group-chart-${group}`">
		<div ref="chartEl" class="history-chart" :style="{ height: `${chartHeight}px` }"></div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import {
	echarts,
	axisNameStyle,
	FONT_FAMILY,
	forecastGrid,
	forecastYAxis,
	lineCasing,
	tooltipStyle,
	tooltipTable,
	xAxisLabelStyle,
	type TooltipRow,
	lineDefaults,
	hoverDot,
} from "../Forecast/echarts";
import colors, {
	resolveColors,
	deviceColorMap,
	darken,
	batteryColor,
	setAlpha,
	lighterColor,
} from "@/colors";
import store from "@/store";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import echartsChart from "@/mixins/echartsChart";
import { PERIODS } from "../Sessions/types";
import { is12hFormat } from "@/units";
import { hasColorPicker, isBidirectional } from "./groups";
import type { PriceBand, PriceOverlay } from "./types";
import { CURRENCY } from "@/types/evcc";
import { energyAxisScale, type EnergyAxisScale } from "@/utils/energyAxis";
import { labelStep, DAY_STEPS, MONTH_STEPS } from "@/utils/labelStep";
import { PANEL_EXTRA, panelGrids, socTempPanel, type SubPanel } from "./subPanel";

export interface HistorySlot {
	start: string;
	end: string;
	energy: number;
	returnEnergy: number;
	socTemp?: number; // a single slot's soc in percent or temperature
	cost?: number; // grid import, where a price exists
	pricedEnergy?: number; // kWh with a price
	returnCost?: number; // grid export, where a price exists
	pricedReturnEnergy?: number; // kWh with a price
}

export interface HistorySeries {
	title: string;
	group: string;
	data: HistorySlot[];
	// Marks a synthetic / derived series (e.g. "other consumers"). Gets a neutral
	// color and is excluded from the source-data table.
	virtual?: boolean;
	// explicit color, skips palette resolution
	color?: string;
	// socTemp holds a temperature, the entity heats instead of charging
	isTemp?: boolean;
	// Stable index into the palette, preserved across navigations even when the
	// displayed list is filtered (e.g. inactive loadpoints dropped) so an
	// entity keeps its color when navigating between periods.
	paletteIndex?: number;
}

type WithChartOption = { chartOption: Record<string, unknown> };

// Alpha (0..1) for entity at position `i` in a `n`-entity stack: top is opaque,
// each step below fades by 20%, capped at 50%.
export function stepAlpha(i: number, n: number): number {
	const step = 0.2;
	const minAlpha = 0.5;
	return Math.max(minAlpha, 1 - (n - 1 - i) * step);
}

// Multiple entities stack into one bar; grid and meter render side-by-side.
const STACKED_GROUPS: ReadonlySet<string> = new Set(["loadpoint", "consumer", "pv", "battery"]);

export default defineComponent({
	name: "GroupChart",
	mixins: [formatter, echartsChart],
	props: {
		group: { type: String, required: true },
		color: { type: String, required: true },
		series: { type: Array as PropType<HistorySeries[]>, default: () => [] },
		overlay: { type: Array as PropType<HistorySeries[]>, default: () => [] },
		overlayColor: { type: String, default: "" },
		overlayLabel: { type: String, default: "" },
		showOverlay: { type: Boolean, default: false },
		focusedEntity: { type: Number as PropType<number | null>, default: null },
		period: { type: String as PropType<PERIODS>, required: true },
		from: { type: Date, required: true },
		to: { type: Date, required: true },
		// one value per row (energy or returnEnergy), no direction columns
		singleValue: Boolean,
		// stack entities regardless of group
		stacked: Boolean,
		// bidirectional data with an automatic range instead of a symmetric one
		autoRange: Boolean,
		height: { type: Number, default: 180 },
		// soc or temperature of the day's slots in a panel below the bars
		socTemp: Boolean,
		// import and export price per category: in the tooltip, and in a panel below
		// the bars when a price moves
		prices: { type: Object as PropType<PriceOverlay | null>, default: null },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
		// the bands below the baseline continue the order above it in reverse, so a
		// stack reads as one column crossing zero instead of two stacks meeting there.
		// Toggling it, like the price overlay, swaps without animation
		stackThrough: Boolean,
		// stacked charts share one axis, only the last one labels it
		showXAxis: { type: Boolean, default: true },
		// what the parts above and below the baseline stand for, written along the left edge
		upperLabel: { type: String, default: "" },
		lowerLabel: { type: String, default: "" },
	},
	emits: ["slot"],
	data(): {
		isMobile: boolean;
		mediaQuery: MediaQueryList | null;
		previousFocusedEntity: number | null;
		previousPeriod: PERIODS;
		previousSeriesKey: string;
		previousView: string;
		activeSlot: number | null;
		chartWidth: number;
	} {
		return {
			isMobile: false,
			mediaQuery: null,
			chartWidth: 0,
			previousFocusedEntity: this.focusedEntity as number | null,
			previousPeriod: this.period as PERIODS,
			previousSeriesKey: "",
			previousView: `${this.stackThrough}-${!!this.prices}`,
			activeSlot: null,
		};
	},
	computed: {
		// In day view a slot is 15 minutes of energy (kWh). Display as
		// average power (kW): 15 min slot ⇒ kW = kWh × 4.
		valueFactor(): number {
			return this.period === PERIODS.DAY ? 4 : 1;
		},
		stackEntities(): boolean {
			return this.stacked || STACKED_GROUPS.has(this.group);
		},
		// Peak of stacked per-slot sums, incl. overlay when shown so its line isn't
		// clipped. Bidirectional: pos/neg separately.
		axisPeak(): number {
			const factor = this.valueFactor;
			// Stacked groups: sum entities per slot. Unstacked (grid, meter): max per entity.
			const peak = (series: HistorySeries[], pick: (slot: HistorySlot) => number) => {
				if (this.stackEntities) {
					const sums = new Map<string, number>();
					for (const s of series) {
						for (const slot of s.data) {
							sums.set(slot.start, (sums.get(slot.start) || 0) + pick(slot) * factor);
						}
					}
					let max = 0;
					for (const v of sums.values()) if (v > max) max = v;
					return max;
				}
				let max = 0;
				for (const s of series) {
					for (const slot of s.data) {
						const v = pick(slot) * factor;
						if (v > max) max = v;
					}
				}
				return max;
			};
			if (this.isBidirectional) {
				return Math.max(
					peak(this.visibleSeries, (slot) => Math.abs(slot.energy)),
					peak(this.visibleSeries, (slot) => Math.abs(slot.returnEnergy))
				);
			}
			const overlay = this.showOverlay ? this.overlay : [];
			return Math.max(
				peak(this.visibleSeries, (slot) => slot.energy),
				peak(overlay, (slot) => slot.energy)
			);
		},
		// axisPeak is in kW(h), the shared scale works in W(h)
		axisScale(): EnergyAxisScale {
			return energyAxisScale(this.axisPeak * 1000);
		},
		useSmallUnit(): boolean {
			return this.axisScale.unit === POWER_UNIT.W;
		},
		axisLimit(): number {
			return this.axisScale.limit / 1000;
		},
		unit(): "W" | "Wh" | "kW" | "kWh" {
			if (this.period === PERIODS.DAY) return this.useSmallUnit ? "W" : "kW";
			return this.useSmallUnit ? "Wh" : "kWh";
		},
		// Picker groups have many entity colors, so use a neutral tooltip background.
		tooltipColor(): string {
			if (hasColorPicker(this.group)) {
				return colors.text || this.color;
			}
			return this.color;
		},
		visibleSeries(): HistorySeries[] {
			if (this.focusedEntity === null) return this.series;
			const idx = this.focusedEntity;
			return this.series.filter((s, i) => (s.paletteIndex ?? i) === idx);
		},
		isBidirectional(): boolean {
			return isBidirectional(this.group, this.series);
		},
		// day view: soc or temperature per category from the first series carrying it
		socTempValues(): (number | null)[] | null {
			if (this.period !== PERIODS.DAY || !this.socTemp) return null;
			const source = this.series.find((s) => s.data.some((slot) => slot.socTemp != null));
			if (!source) return null;
			const out: (number | null)[] = this.categoryKeys.map(() => null);
			for (const slot of source.data) {
				const idx = this.categoryKeys.indexOf(
					this.timestampKey(new Date(slot.start).getTime())
				);
				if (idx >= 0) out[idx] = slot.socTemp ?? null;
			}
			return out;
		},
		// below the bars: soc or temperature of the day as line and area, or the grid prices
		subPanel(): SubPanel | null {
			if (!this.socTempValues) return this.pricePanel;
			const isTemp = this.socTempIsTemp;
			return socTempPanel(
				this.socTempValues,
				this.entryColors[0] || this.color,
				isTemp,
				(v) => (isTemp ? this.fmtNumber(v, 0, "celsius") : this.fmtPercentage(v))
			);
		},
		chartHeight(): number {
			return this.height + (this.subPanel ? PANEL_EXTRA : 0);
		},
		// the baseline splits the plot, a label for each side that has data. Pinned to
		// the plot's corners so they stay put between periods: read bottom up, the
		// upper label ends at the top, the lower one starts at the bottom
		sideLabels(): Record<string, unknown>[][] {
			const has = (key: "energy" | "returnEnergy") =>
				this.series.some((s) => s.data.some((slot) => slot[key] > 0));
			// each area runs from the baseline to its axis end (an infinite value is
			// clipped to it) and carries its label on the outer corner: as far left of the
			// plot as the tick labels are right of it, and shifted along the rotated text
			// (positive is up) to the chart's own top and bottom
			return [
				...(this.upperLabel && has("energy")
					? [
							[
								{
									name: this.upperLabel,
									yAxis: 0,
									label: {
										position: [-14, "0%"],
										align: "right",
										offset: [28, 0],
									},
								},
								{ yAxis: Infinity },
							],
						]
					: []),
				...(this.lowerLabel && has("returnEnergy")
					? [
							[
								{
									name: this.lowerLabel,
									yAxis: 0,
									label: {
										position: [-14, "100%"],
										align: "left",
										offset: [-17, 0],
									},
								},
								{ yAxis: -Infinity },
							],
						]
					: []),
			];
		},
		// tooltip rows by stable entity index. A stack reads top down: what stacks
		// upwards in reverse series order, then what hangs below the baseline, which
		// continues that order when stacked through
		rowOrder(): number[] {
			const rows = this.series.map((s, i) => ({
				idx: s.paletteIndex ?? i,
				up: s.data.some((slot) => slot.energy > 0),
			}));
			if (!this.stackEntities) return rows.map((r) => r.idx);
			const below = rows.filter((r) => !r.up);
			return [
				...rows.filter((r) => r.up).reverse(),
				...(this.stackThrough ? below.reverse() : below),
			].map((r) => r.idx);
		},
		// another view of the same data: stack order or the price overlay toggled
		viewKey(): string {
			return `${this.stackThrough}-${!!this.prices}`;
		},
		// the prices that move over the period, zoomed into their range. A constant price
		// is only in the tooltip
		pricePanel(): SubPanel | null {
			const known = (v: (number | null)[]) => v.filter((x): x is number => x !== null);
			const lo = (band: PriceBand) => Math.min(...known(band.lo));
			const hi = (band: PriceBand) => Math.max(...known(band.hi));
			const bands: { key: string; band: PriceBand; color: string }[] = [];
			const add = (key: string, band: PriceBand | undefined, color: string) => {
				// an effective price carries float noise, a flat tariff must stay flat
				if (band && hi(band) - lo(band) > 1e-4) bands.push({ key, band, color });
			};
			add("import", this.prices?.import, colors.price || "");
			add("feedin", this.prices?.feedin, colors.export || "");
			if (!bands.length) return null;
			const min = Math.min(...bands.map(({ band }) => lo(band)));
			const max = Math.max(...bands.map(({ band }) => hi(band)));
			const series = ({ key, band, color }: (typeof bands)[number]) => ({
				id: `price-${key}`,
				name: `price-${key}`,
				type: "line",
				xAxisIndex: 1,
				yAxisIndex: 1,
				data: band.avg,
				// a price holds for its whole slot: points sit at slot centers, so the
				// jump between two points lands on the slot boundary
				step: "middle",
				...hoverDot(color),
				lineStyle: { ...lineDefaults, color },
				// glow fading out below the line, like the price forecast
				areaStyle: {
					color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
						{ offset: 0, color: lighterColor(color) || color },
						{ offset: 0.75, color: setAlpha(color, "00") || color },
						{ offset: 1, color: setAlpha(color, "00") || color },
					]),
				},
				z: 4,
			});
			return {
				track: "transparent",
				series: bands.map(series),
				// the exact price range, currencies differ too much for rounding rules
				yAxis: forecastYAxis({
					gridIndex: 1,
					position: "right",
					min,
					max,
					// labels at both ends only
					interval: max - min,
					splitLine: { show: false },
					axisLabel: {
						color: colors.muted || "",
						hideOverlap: true,
						formatter: (v: number) => this.fmtPricePerKWh(v, this.currency, true),
					},
				}),
			};
		},
		socTempIsTemp(): boolean {
			return !!this.series.find((s) => s.data.some((slot) => slot.socTemp != null))?.isTemp;
		},
		categoryTimestamps(): number[] {
			const out: number[] = [];
			const cursor = new Date(this.from);
			const end = this.to.getTime();
			if (this.period === PERIODS.YEAR) {
				while (cursor.getTime() < end) {
					out.push(cursor.getTime());
					cursor.setMonth(cursor.getMonth() + 1);
				}
			} else if (this.period === PERIODS.MONTH) {
				while (cursor.getTime() < end) {
					out.push(cursor.getTime());
					cursor.setDate(cursor.getDate() + 1);
				}
			} else {
				// DAY → 15-minute slots
				while (cursor.getTime() < end) {
					out.push(cursor.getTime());
					cursor.setMinutes(cursor.getMinutes() + 15);
				}
			}
			return out;
		},
		// Per-period stable keys so echarts treats positions as the SAME category
		// across navigations (year→year: 12 month keys; day→day: 96 slot keys),
		// which lets the diff animate value transitions instead of replacing bars.
		categoryKeys(): string[] {
			return this.categoryTimestamps.map((t) => this.timestampKey(t));
		},
		// Which category slots carry a bar, so hover can skip empty slots.
		slotsWithData(): boolean[] {
			const index = new Map(this.categoryKeys.map((k, i) => [k, i]));
			const has = Array.from({ length: this.categoryKeys.length }, () => false);
			for (const s of this.visibleSeries) {
				for (const slot of s.data) {
					if (slot.energy <= 0 && (!this.isBidirectional || slot.returnEnergy <= 0))
						continue;
					const idx = index.get(this.timestampKey(new Date(slot.start).getTime()));
					if (idx !== undefined) has[idx] = true;
				}
			}
			return has;
		},
		entryColors(): string[] {
			// Picker groups color per entity; pv/battery use darker steps of the group color.
			if (hasColorPicker(this.group)) {
				const mutedColor = colors.muted || this.color;
				const titles: string[] = [];
				for (const s of this.series) {
					if (!s.virtual && !titles.includes(s.title)) titles.push(s.title);
				}
				const palette = resolveColors(titles, deviceColorMap(store.state.deviceColors));
				return this.series.map((s) => {
					if (s.color) return s.color;
					// Virtual "other consumers" entity renders in a neutral gray to set
					// it apart from explicit meter entities.
					if (s.virtual) return mutedColor;
					return palette[s.title] || this.color;
				});
			}
			if (this.group === "battery") {
				return this.series.map((s, i) => batteryColor(s.paletteIndex ?? i));
			}
			if (this.series.length <= 1) return [this.color];
			return this.series.map((_, i) => darken(this.color, stepAlpha(i, this.series.length)));
		},
		echartsSeries() {
			const cats = this.categoryKeys;
			const index = new Map<string, number>();
			cats.forEach((k, i) => index.set(k, i));
			const slotKey = (start: string) => this.timestampKey(new Date(start).getTime());
			const radius = 2;
			const factor = this.valueFactor;

			const result: Record<string, unknown>[] = [];

			// Always render overlay slot (line series) so series structure is stable;
			// data is all-null when toggled off.
			const overlayValues: (number | null)[] = Array.from(
				{ length: cats.length },
				() => null
			);
			if (this.showOverlay && this.overlay.length) {
				for (const s of this.overlay) {
					for (const slot of s.data) {
						const idx = index.get(slotKey(slot.start));
						if (idx === undefined) continue;
						overlayValues[idx] = (overlayValues[idx] || 0) + slot.energy * factor;
					}
				}
			}
			const overlayCol = this.overlayColor || this.color;
			const overlay = {
				id: "overlay",
				name: this.overlayLabel || "overlay",
				type: "line",
				data: overlayValues,
				smooth: true,
				symbol: "none",
				connectNulls: true,
				lineStyle: { color: overlayCol, ...lineDefaults },
				itemStyle: { color: overlayCol },
				z: 4,
			};
			// the side labels ride on the overlay, not on its casing
			result.push(lineCasing(overlay, 3), {
				...overlay,
				...(this.sideLabels.length
					? {
							// invisible areas above and below the baseline, each carrying its label
							markArea: {
								silent: true,
								itemStyle: { color: "transparent" },
								label: {
									rotate: 90,
									verticalAlign: "middle",
									color: colors.muted || "",
									fontSize: 10,
									opacity: 0.75,
								},
								data: this.sideLabels,
							},
						}
					: {}),
			});

			if (this.subPanel) result.push(...this.subPanel.series);

			// Always render import + export series per entity, even if one direction
			// is empty (null-filled). Stable series ids/structure across renders so
			// echarts can animate value transitions instead of redrawing from zero.
			// Build value arrays per entity first so we can determine, per slot,
			// which entity is the *visible* top/bottom of the stack — that one
			// gets the rounded cap even if higher-index entities are zero/null.
			const energyByEntity: (number | null)[][] = [];
			const returnEnergyByEntity: (number | null)[][] = [];
			this.series.forEach((s, i) => {
				const energyValues: (number | null)[] = Array.from(
					{ length: cats.length },
					() => null
				);
				const returnEnergyValues: (number | null)[] = Array.from(
					{ length: cats.length },
					() => null
				);
				const hidden =
					this.focusedEntity !== null && this.focusedEntity !== (s.paletteIndex ?? i);
				if (!hidden) {
					for (const slot of s.data) {
						const idx = index.get(slotKey(slot.start));
						if (idx === undefined) continue;
						if (slot.energy > 0) energyValues[idx] = slot.energy * factor;
						// non-bidirectional groups ignore return energy
						if (this.isBidirectional && slot.returnEnergy > 0)
							returnEnergyValues[idx] = -slot.returnEnergy * factor;
					}
				}
				energyByEntity.push(energyValues);
				returnEnergyByEntity.push(returnEnergyValues);
			});
			// Per slot: index of the topmost (largest i) entity with a non-zero
			// value. -1 = no entity has data at that slot.
			const topEnergyPerSlot: number[] = Array.from({ length: cats.length }, () => -1);
			const topReturnEnergyPerSlot: number[] = Array.from({ length: cats.length }, () => -1);
			for (let i = 0; i < this.series.length; i++) {
				for (let idx = 0; idx < cats.length; idx++) {
					if ((energyByEntity[i]![idx] ?? 0) > 0) topEnergyPerSlot[idx] = i;
					// stacked through, the first entity with data is the deepest
					if (
						(returnEnergyByEntity[i]![idx] ?? 0) < 0 &&
						(!this.stackThrough || topReturnEnergyPerSlot[idx] === -1)
					)
						topReturnEnergyPerSlot[idx] = i;
				}
			}

			// hover lightens the active slot's bars (onChartMouseMove), like the sessions
			// charts; silent stops single-segment highlight
			const barEmphasis = { silent: true };
			const returnSeries: Record<string, unknown>[] = [];
			this.series.forEach((s, i) => {
				const c = this.entryColors[i] || this.color;
				const returnEnergyColor =
					(s.group === "grid" && colors.export) ||
					(s.group === "battery" ? setAlpha(c, "cc") || c : c);
				const energyValues = energyByEntity[i]!;
				const returnEnergyValues = returnEnergyByEntity[i]!;
				const energyName =
					this.series.length > 1 || this.isBidirectional
						? this.directionLabel(s, "energy")
						: this.singleEntityName(s);
				const returnEnergyName = this.directionLabel(s, "returnEnergy");
				// Same stack name for import and export means they share one x slot
				// (positive values stack up, negative stack down, no width penalty).
				const stackName = this.stackEntities ? `group-${this.group}` : `entity-${i}`;
				// Rounded cap goes on the visible top/bottom per slot: non-stacked bars
				// always cap; stacked groups cap the topmost non-zero entity so an empty
				// top entity doesn't drop the rounding; a focused entity is solo.
				const stableIdx = s.paletteIndex ?? i;
				const energyData: (
					| number
					| null
					| { value: number; itemStyle: { borderRadius: number[] } }
				)[] = energyValues.map((v, idx) => {
					if (v == null) return v;
					const isTop =
						!this.stackEntities ||
						topEnergyPerSlot[idx] === i ||
						this.focusedEntity === stableIdx;
					if (!isTop) return v;
					return { value: v, itemStyle: { borderRadius: [radius, radius, 0, 0] } };
				});
				const returnEnergyData: (
					| number
					| null
					| { value: number; itemStyle: { borderRadius: number[] } }
				)[] = returnEnergyValues.map((v, idx) => {
					if (v == null) return v;
					const isBottom =
						!this.stackEntities ||
						topReturnEnergyPerSlot[idx] === i ||
						this.focusedEntity === stableIdx;
					if (!isBottom) return v;
					return { value: v, itemStyle: { borderRadius: [0, 0, radius, radius] } };
				});
				result.push({
					id: `entity-${stableIdx}-energy`,
					name: energyName,
					type: "bar",
					stack: stackName,
					data: energyData,
					itemStyle: { color: c, borderRadius: [0, 0, 0, 0] },
					barCategoryGap: "25%",
					barGap: "10%",
					...barEmphasis,
				});
				(this.stackThrough ? returnSeries : result).push({
					id: `entity-${stableIdx}-returnEnergy`,
					name: returnEnergyName,
					type: "bar",
					stack: stackName,
					data: returnEnergyData,
					itemStyle: { color: returnEnergyColor, borderRadius: [0, 0, 0, 0] },
					barCategoryGap: "25%",
					barGap: "10%",
					...barEmphasis,
				});
			});
			result.push(...returnSeries.reverse());

			return result;
		},
		labelStep(): number {
			if (this.period === PERIODS.DAY) {
				// wider "4 PM" labels need more room
				return labelStep(24, this.chartWidth, DAY_STEPS, is12hFormat() ? 56 : 40);
			}
			return labelStep(31, this.chartWidth, MONTH_STEPS);
		},
		labelForTimestamp(): (t: number) => string {
			if (this.period === PERIODS.DAY) {
				// Skip 00:00 so the chart can align with the section title on the left.
				const stepHours = this.labelStep;
				return (t: number) => {
					const d = new Date(t);
					if (d.getMinutes() !== 0) return "";
					const h = d.getHours();
					if (h === 0 || h % stepHours !== 0) return "";
					return this.hourShort(d);
				};
			}
			if (this.period === PERIODS.MONTH) {
				const step = this.labelStep;
				return (t: number) => {
					const day = new Date(t).getDate();
					return (day - 1) % step === 0 ? `${day}` : "";
				};
			}
			// YEAR — narrow (single-letter) on mobile, short month name otherwise
			return this.isMobile
				? (t: number) => this.fmtMonthNarrow(new Date(t))
				: (t: number) => this.fmtMonth(new Date(t), true);
		},
		// Column headers for bidirectional tooltips (grid: imported/exported,
		// battery: charged/discharged, meter: energy/reverse). Null when the group
		// has no direction labels.
		directionHeaders(): string[] | null {
			if (!this.isBidirectional) return null;
			const energyKey = `energy.direction.${this.group}.energy`;
			const returnEnergyKey = `energy.direction.${this.group}.returnEnergy`;
			const energy = this.$t(energyKey);
			const returnEnergy = this.$t(returnEnergyKey);
			if (energy === energyKey || returnEnergy === returnEnergyKey) return null;
			return [String(energy), String(returnEnergy)];
		},
		tooltipDateLabel(): (t: number) => string {
			if (this.period === PERIODS.DAY) {
				return (t) => this.fmtTimeSlot(new Date(t), 15 * 60 * 1000);
			}
			if (this.period === PERIODS.MONTH) {
				return (t) => this.fmtDayMonth(new Date(t));
			}
			return (t) => this.fmtMonthYear(new Date(t));
		},
		chartOption(): Record<string, unknown> {
			const cats = this.categoryTimestamps;
			const keys = this.categoryKeys;
			const formatLabel = this.labelForTimestamp;
			const tooltipDate = this.tooltipDateLabel;
			const barGrid = {
				...forecastGrid(),
				left: this.socTempValues ? 36 : this.upperLabel || this.lowerLabel ? 22 : 0,
				right: 36,
				...(this.showXAxis ? {} : { bottom: 4 }),
			};
			const xAxisLabel = {
				...xAxisLabelStyle(),
				show: this.showXAxis,
				hideOverlap: false,
				interval: 0,
				formatter: (_value: string, index: number) => formatLabel(cats[index] ?? 0),
			};
			const barXAxis = {
				type: "category",
				data: keys,
				axisLine: this.isBidirectional
					? {
							show: true,
							onZero: true,
							lineStyle: { color: colors.muted || "", width: 1 },
						}
					: { show: false },
				axisTick: { show: false },
				splitLine: { show: false },
				// with a panel the labels sit below it
				axisLabel: this.subPanel ? { show: false } : xAxisLabel,
			};
			return {
				animation: true,
				animationDuration: 0,
				animationDurationUpdate: 400,
				textStyle: { fontFamily: FONT_FAMILY },
				// the bars stay grid 0, the panel sits below them as grid 1
				grid: this.subPanel ? panelGrids(barGrid, this.subPanel.track) : barGrid,
				// one pointer and tooltip across both grids
				axisPointer: { link: [{ xAxisIndex: "all" }] },
				tooltip: {
					trigger: "axis",
					// transparent shadow snaps to slots without a band; triggerEmphasis off, the slot is highlighted in onChartMouseMove
					axisPointer: {
						type: "shadow",
						triggerEmphasis: false,
						shadowStyle: { color: "transparent" },
					},
					...tooltipStyle(this.tooltipColor),
					formatter: (
						params: {
							value: number | null;
							seriesName: string;
							seriesId: string;
							dataIndex: number;
						}[]
					) => {
						if (!params?.length) return "";
						const hasData = params.some((p) => p.value != null);
						if (!hasData) return "";
						const first = params.find((p) => p.dataIndex != null);
						if (!first) return "";
						const ts = cats[first.dataIndex];
						const head = ts != null ? tooltipDate(ts) : "";

						// Collect energy/returnEnergy values per entity from this slot's params.
						const totals = new Map<number, { energy: number; returnEnergy: number }>();
						for (const p of params) {
							const m = /^entity-(\d+)-(energy|returnEnergy)$/.exec(p.seriesId || "");
							if (!m) continue;
							const i = parseInt(m[1] || "", 10);
							const t = totals.get(i) ?? { energy: 0, returnEnergy: 0 };
							const v = Math.abs(p.value ?? 0);
							if (m[2] === "energy") t.energy = v;
							else t.returnEnergy = v;
							totals.set(i, t);
						}
						// Always list every visible entity, even when its values for
						// this slot are zero or missing — keeps the tooltip layout
						// stable across slots.
						const indices =
							this.focusedEntity !== null ? [this.focusedEntity] : this.rowOrder;
						const nameByIdx = new Map(
							this.series.map((s, i) => [s.paletteIndex ?? i, s.title])
						);
						const showName = this.series.length > 1 && this.focusedEntity === null;

						// one unit for all rows, based on the largest individual value (not the total)
						const rowValues = indices.map(
							(i) => totals.get(i) ?? { energy: 0, returnEnergy: 0 }
						);
						const unit = this.getPowerUnit(
							Math.max(
								0,
								...rowValues.flatMap((t) =>
									this.isBidirectional && !this.singleValue
										? [t.energy, t.returnEnergy]
										: [t.energy + t.returnEnergy]
								)
							) * 1000
						);
						const formatValue = (v: number) => {
							const watts = Math.abs(v) * 1000;
							return this.period === PERIODS.DAY
								? this.fmtW(watts, unit)
								: this.fmtWh(watts, unit);
						};

						// the overlay line's value in this slot, e.g. the solar forecast
						const forecast = params.find((p) => p.seriesId === "overlay")?.value;
						const rows: TooltipRow[] = indices.map((i, idx) => {
							const t = rowValues[idx] ?? { energy: 0, returnEnergy: 0 };
							// a single value covers both sides, a sink can draw from above and below
							const values =
								this.isBidirectional && !this.singleValue
									? [formatValue(t.energy), formatValue(t.returnEnergy)]
									: [formatValue(t.energy + t.returnEnergy)];
							return {
								// the price or forecast row below needs a name column to line up with
								name: showName
									? (nameByIdx.get(i) ?? "")
									: this.prices
										? this.$t("energy.grid.energy")
										: forecast != null
											? this.singleEntityName(this.series[idx]!)
											: undefined,
								values,
							};
						});
						// cost or revenue and the price of the slot, per direction
						const at = (
							values: (number | null)[] | undefined,
							fmt: (v: number) => string
						) => {
							const v = values?.[first.dataIndex];
							return v == null ? "" : fmt(v);
						};
						if (this.prices) {
							const { import: imported, feedin } = this.prices;
							const money = (v: number) => this.fmtMoneyWithSymbol(v, this.currency);
							const price = (v: number) => this.fmtPricePerKWh(v, this.currency);
							const amounts = [at(imported?.cost, money), at(feedin?.cost, money)];
							if (amounts.some((v) => v !== "")) {
								rows.push({ name: this.$t("energy.grid.amount"), values: amounts });
							}
							rows.push({
								// a slot has one price, a longer bucket the average paid
								name:
									this.period === PERIODS.DAY
										? this.$t("energy.grid.price")
										: `ø ${this.$t("energy.grid.price")}`,
								values: [at(imported?.avg, price), at(feedin?.avg, price)],
							});
						}
						if (showName) {
							const sum = (key: "energy" | "returnEnergy") =>
								rowValues.reduce((acc, t) => acc + t[key], 0);
							rows.push({
								name: this.$t("sessions.total"),
								values:
									this.isBidirectional && !this.singleValue
										? [
												formatValue(sum("energy")),
												formatValue(sum("returnEnergy")),
											]
										: [formatValue(sum("energy") + sum("returnEnergy"))],
								total: true,
							});
						}
						// below the separator, it is not part of the sum
						if (forecast != null) {
							rows.push({
								name: this.overlayLabel,
								values: [formatValue(forecast)],
								total: true,
							});
						}
						// one entity with its soc or temperature in the day view: a named row
						// per value
						const socTemp = this.socTempValues?.[first.dataIndex];
						if (socTemp != null && !showName) {
							const socTempText = this.socTempIsTemp
								? this.fmtTemperature(socTemp)
								: this.fmtPercentage(socTemp);
							const t = rowValues[0] ?? { energy: 0, returnEnergy: 0 };
							const label = (key: string) => this.$t(`energy.socTemp.${key}`);
							const named: TooltipRow[] = this.isBidirectional
								? [
										{
											name: this.directionHeaders?.[0],
											values: [formatValue(t.energy)],
										},
										{
											name: this.directionHeaders?.[1],
											values: [formatValue(t.returnEnergy)],
										},
									]
								: [
										{
											name: label(this.socTempIsTemp ? "used" : "charged"),
											values: [formatValue(t.energy)],
										},
									];
							named.push({
								name: label(this.socTempIsTemp ? "temperature" : "soc"),
								values: [socTempText],
							});
							return tooltipTable(head, named);
						}
						return tooltipTable(
							head,
							rows,
							this.singleValue ? undefined : (this.directionHeaders ?? undefined)
						);
					},
				},
				xAxis: this.subPanel
					? [
							barXAxis,
							{
								...barXAxis,
								gridIndex: 1,
								axisLine: { show: false },
								axisLabel: xAxisLabel,
							},
						]
					: barXAxis,
				yAxis: [
					forecastYAxis({
						// automatic range must be allowed below zero for the export band
						...(this.autoRange && this.isBidirectional ? { min: undefined } : {}),
						...(this.isBidirectional && this.axisLimit > 0 && !this.autoRange
							? {
									min: -this.axisLimit,
									max: this.axisLimit,
									interval: this.axisLimit / 2,
								}
							: this.useSmallUnit
								? { max: this.axisLimit, interval: this.axisLimit / 4 }
								: {}),
						position: "right",
						splitNumber: 3,
						splitLine: {
							showMinLine: true,
							showMaxLine: true,
							lineStyle: { color: colors.border || "" },
						},
						name: this.unit,
						...axisNameStyle(),
						axisLabel: {
							color: colors.muted || "",
							hideOverlap: true,
							formatter: (v: number): string => {
								const { unit, digits } = this.axisScale;
								return this.period === PERIODS.DAY
									? this.fmtW(v * 1000, unit, false, digits)
									: this.fmtWh(v * 1000, unit, false, digits);
							},
						},
					}),
					...(this.subPanel ? [this.subPanel.yAxis] : []),
				],
				series: this.echartsSeries,
			};
		},
	},
	mounted() {
		const zr = this.chart?.getZr();
		zr?.on("mousemove", this.onChartMouseMove);
		zr?.on("globalout", this.clearHighlight);
		this.mediaQuery = window.matchMedia("(max-width: 575.98px)");
		this.isMobile = this.mediaQuery.matches;
		this.mediaQuery.addEventListener("change", this.onMediaChange);
	},
	beforeUnmount() {
		this.mediaQuery?.removeEventListener("change", this.onMediaChange);
	},
	methods: {
		onChartInit() {
			this.chartWidth = this.chart?.getWidth() ?? 0;
			// whole column is clickable, not only the bar itself
			this.chart?.getZr().on("click", (e: { offsetX: number; offsetY: number }) => {
				this.emitSlotAt(e.offsetX, e.offsetY);
			});
			// echarts lifts a highlighted element above its siblings, the svg renderer
			// then re-inserts every bar on each hover step. Nothing overlaps here, so
			// the order can stay
			this.chart?.on("finished", () => {
				const elements = this.chart?.getZr().storage.getDisplayList() ?? [];
				for (const el of elements) (el as { z2EmphasisLift?: number }).z2EmphasisLift = 0;
			});
		},
		onChartResize() {
			this.chartWidth = this.chart?.getWidth() ?? 0;
		},
		onChartTap(x: number, y: number) {
			this.emitSlotAt(x, y);
		},
		// inside the bars or the panel below them, both share the x axis
		inPlot(point: number[]): boolean {
			const grids = this.subPanel ? [0, 1] : [0];
			return grids.some((gridIndex) => !!this.chart?.containPixel({ gridIndex }, point));
		},
		emitSlotAt(x: number, y: number) {
			const chart = this.chart;
			if (!chart || !this.inPlot([x, y])) return;
			const [idx] = chart.convertFromPixel({ seriesIndex: 0 }, [x, y]);
			const start = this.categoryTimestamps[Math.round(idx ?? -1)];
			if (start !== undefined) this.$emit("slot", new Date(start));
		},
		applyChartOption() {
			// the element may have been laid out after init
			this.chartWidth = this.chart?.getWidth() ?? 0;
			const opt = (this as unknown as WithChartOption).chartOption;
			const focusChanged = this.previousFocusedEntity !== this.focusedEntity;
			const periodChanged = this.previousPeriod !== this.period;
			// Fingerprint the set of series IDs in their render order so we can
			// detect when entities are added or removed (e.g. a filtered loadpoint
			// re-appears after navigating to a new day).
			const newSeriesKey = (opt["series"] as Array<{ id?: string }>)
				.map((s) => s.id ?? "")
				.join(",");
			// Full reset on period/composition change — replaceMerge re-appends
			// re-introduced series at the end and flips stack order. Otherwise
			// partial update lets stable IDs animate value transitions.
			const viewChanged = this.previousView !== this.viewKey;
			const fullReset =
				periodChanged || viewChanged || newSeriesKey !== this.previousSeriesKey;
			this.chart?.setOption(
				fullReset
					? { ...opt, animation: !viewChanged }
					: {
							animation: !focusChanged,
							xAxis: opt["xAxis"],
							yAxis: opt["yAxis"],
							series: opt["series"],
							tooltip: opt["tooltip"],
						},
				fullReset ? { notMerge: true } : { replaceMerge: ["series", "yAxis"] }
			);
			this.previousFocusedEntity = this.focusedEntity as number | null;
			this.previousPeriod = this.period as PERIODS;
			this.previousSeriesKey = newSeriesKey;
			this.previousView = this.viewKey;
		},
		onTouchTooltipReset() {
			this.clearHighlight();
		},
		// highlight the hovered slot. Manual, so slots without a bar are skipped and the panel's line gets its dot
		onChartMouseMove(e: { offsetX: number; offsetY: number }) {
			if (!this.chart) return;
			const point: [number, number] = [e.offsetX, e.offsetY];
			if (!this.inPlot(point)) {
				this.clearHighlight();
				return;
			}
			const grid = this.chart.convertFromPixel({ gridIndex: 0 }, point) as number[];
			const slot = Math.round(grid[0]!);
			if (slot === this.activeSlot) return;
			// nothing to highlight in a slot without a bar, the panel's line still gets
			// its dot there
			if (!this.slotsWithData[slot] && !this.subPanel) {
				this.clearHighlight();
				return;
			}
			this.activeSlot = slot;
			this.chart.dispatchAction({ type: "downplay" });
			this.chart.dispatchAction({
				type: "highlight",
				dataIndex: slot,
				...(this.slotsWithData[slot]
					? {}
					: { seriesId: this.subPanel?.series.map((s) => s["id"]) }),
			});
		},
		clearHighlight() {
			if (this.activeSlot === null) return;
			this.activeSlot = null;
			this.chart?.dispatchAction({ type: "downplay" });
		},
		onMediaChange(e: MediaQueryListEvent) {
			this.isMobile = e.matches;
		},
		timestampKey(t: number): string {
			const d = new Date(t);
			if (this.period === PERIODS.YEAR) return `m${d.getMonth()}`;
			if (this.period === PERIODS.MONTH) return `d${d.getDate()}`;
			return `t${d.getHours()}:${d.getMinutes()}`;
		},
		directionLabel(s: HistorySeries, dir: "energy" | "returnEnergy"): string {
			const key = `energy.direction.${s.group}.${dir}`;
			const label = this.$t(key);
			if (label === key) return s.title;
			if (this.series.length > 1) return `${s.title} ${label}`;
			return String(label);
		},
		singleEntityName(s: HistorySeries): string {
			if (this.series.length > 1) return s.title;
			if (s.group === "loadpoint") {
				return this.$t(s.isTemp ? "energy.flow.heating" : "energy.flow.charging");
			}
			const key = `energy.group.${s.group}`;
			const label = this.$t(key);
			return label === key ? s.title : String(label);
		},
	},
});
</script>

<style scoped>
.history-chart-wrapper {
	width: 100%;
}
.history-chart {
	width: 100%;
}
</style>
