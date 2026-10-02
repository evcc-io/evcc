<template>
	<div class="history-chart-wrapper" :data-testid="`group-chart-${group}`">
		<div ref="chartEl" class="history-chart" :style="{ height: `${height}px` }"></div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import {
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
} from "../Forecast/echarts";
import colors, { resolveColors, deviceColorMap, darken, batteryColor, setAlpha } from "@/colors";
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

export interface HistorySlot {
	start: string;
	end: string;
	energy: number;
	returnEnergy: number;
	socTemp?: number; // a single slot's soc in percent or temperature
	socTempMin?: number; // range over an aggregated bucket
	socTempMax?: number;
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

// `pick` per slot of the first series carrying it, mapped onto the categories
function perCategory<T>(
	series: HistorySeries[],
	keys: string[],
	keyOf: (t: number) => string,
	pick: (slot: HistorySlot) => T | null
): (T | null)[] | null {
	const source = series.find((s) => s.data.some((slot) => pick(slot) !== null));
	if (!source) return null;
	const index = new Map(keys.map((k, i) => [k, i]));
	const out: (T | null)[] = keys.map(() => null);
	for (const slot of source.data) {
		const idx = index.get(keyOf(new Date(slot.start).getTime()));
		if (idx !== undefined) out[idx] = pick(slot);
	}
	return out;
}

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
		// soc or temperature on its own scale: the slots' line in the day view, the
		// bucket's range as a band per column otherwise
		socTemp: Boolean,
		// import and export price per category on a left axis. Lines in the day view,
		// the slot range as a stepped band otherwise
		prices: { type: Object as PropType<PriceOverlay | null>, default: null },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
		// the bands below the baseline continue the order above it in reverse, so a
		// stack reads as one column crossing zero instead of two stacks meeting there.
		// Toggling it, like the price overlay, swaps without animation
		stackThrough: Boolean,
		// stacked charts share one axis, only the last one labels it
		showXAxis: { type: Boolean, default: true },
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
			if (this.period !== PERIODS.DAY) return null;
			return this.socTemp
				? perCategory(
						this.series,
						this.categoryKeys,
						this.timestampKey,
						(slot) => slot.socTemp ?? null
					)
				: null;
		},
		// month and year: min and max of soc or temperature per category
		socTempBands(): ([number, number] | null)[] | null {
			if (this.period === PERIODS.DAY) return null;
			return this.socTemp
				? perCategory(this.series, this.categoryKeys, this.timestampKey, (slot) =>
						slot.socTempMin != null && slot.socTempMax != null
							? ([slot.socTempMin, slot.socTempMax] as [number, number])
							: null
					)
				: null;
		},
		// another view of the same data: stack order or the price overlay toggled
		viewKey(): string {
			return `${this.stackThrough}-${!!this.prices}`;
		},
		// highest price of either direction
		priceExtent(): number {
			const top = (band?: PriceBand) =>
				Math.max(0, ...(band?.hi ?? []).filter((v): v is number => v !== null));
			return Math.max(top(this.prices?.import), top(this.prices?.feedin));
		},
		socTempIsTemp(): boolean {
			return !!this.series.find((s) =>
				s.data.some((slot) => slot.socTemp != null || slot.socTempMax != null)
			)?.isTemp;
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
			result.push(lineCasing(overlay, 3), overlay);

			const priceLine = (id: string, data: (number | null)[], color: string) => ({
				id,
				name: id,
				type: "line",
				yAxisIndex: 1,
				data,
				// a price holds from its slot's start: points sit at slot centers, so the
				// jump between two points lands on the slot boundary
				step: this.period === PERIODS.DAY ? "middle" : false,
				symbol: "none",
				lineStyle: { ...lineDefaults, color, width: 1.5 },
				z: 4,
			});
			const priceBand = (key: "import" | "feedin", band: PriceBand, color: string) => {
				if (this.period === PERIODS.DAY) {
					// in front of the bars with a casing, like the forecast line
					const line = priceLine(`price-${key}`, band.avg, color);
					return [lineCasing(line, 3), line];
				}
				// the slot's price range as a block over the whole slot width, a hairline
				// when it is a single price
				return [
					{
						id: `price-${key}`,
						name: `price-${key}`,
						type: "custom",
						yAxisIndex: 1,
						silent: true,
						// behind the bars
						z: 1,
						data: band.lo.flatMap((lo, i) =>
							lo === null || band.hi[i] === null ? [] : [[i, lo, band.hi[i]]]
						),
						renderItem: (
							_: unknown,
							api: {
								value: (i: number) => number;
								coord: (v: number[]) => number[];
								size: (v: number[]) => number[];
							}
						) => {
							const [x, top] = api.coord([api.value(0), api.value(2)]);
							const [, bottom] = api.coord([api.value(0), api.value(1)]);
							const width = api.size([1, 0])[0]!;
							return {
								type: "rect",
								shape: {
									x: x! - width / 2,
									y: top,
									width,
									height: Math.max(1, bottom! - top!),
								},
								style: { fill: setAlpha(color, "33") },
								blur: { style: { opacity: 1 } },
							};
						},
					},
				];
			};
			if (this.prices?.import)
				result.push(...priceBand("import", this.prices.import, colors.price || ""));
			if (this.prices?.feedin)
				result.push(...priceBand("feedin", this.prices.feedin, colors.export || ""));

			if (this.socTempBands) {
				// the range as one path per run of buckets: flat across each slot along the
				// highs, an s-curve over the slot edge to the next, back along the lows
				const runs: { x: number; lo: number; hi: number }[][] = [];
				this.socTempBands.forEach((band, i) => {
					if (!band) return;
					const run = this.socTempBands![i - 1] ? runs.at(-1)! : [];
					if (!run.length) runs.push(run);
					// at least a sliver for a constant value, so the slot still reads as covered
					run.push({ x: i, lo: band[0], hi: Math.max(band[1], band[0] + 1) });
				});
				result.push({
					id: "soctemp",
					name: "soctemp",
					type: "custom",
					yAxisIndex: 1,
					silent: true,
					// one item per run, not per bucket, so it must stay out of the axis tooltip
					tooltip: { show: false },
					z: 1,
					data: runs.map((_, i) => i),
					renderItem: (
						params: { dataIndex: number },
						api: { coord: (v: number[]) => number[]; size: (v: number[]) => number[] }
					) => {
						const run = runs[params.dataIndex]!;
						// slot edges in pixels from the slot's center and width, category
						// coordinates only resolve whole indices
						const width = api.size([1, 0])[0]!;
						const P = (slot: number, offset: number, y: number) => {
							const [cx, cy] = api.coord([slot, y]);
							return `${(cx! + offset * width).toFixed(1)},${cy!.toFixed(1)}`;
						};
						// flat within a slot up to 0.4 widths from its center, the step eased
						// over the remaining 0.2 around the slot edge; the run's outer edges are
						// square. `sign` walks the highs forward and the lows back
						const edge = (pts: typeof run, key: "lo" | "hi", sign: 1 | -1) =>
							pts
								.map((pt, j) => {
									const next = pts[j + 1];
									const to = ` L${P(pt.x, sign * (next ? 0.4 : 0.5), pt[key])}`;
									return next
										? `${to} C${P(pt.x, sign * 0.5, pt[key])} ${P(next.x, -sign * 0.5, next[key])} ${P(next.x, -sign * 0.4, next[key])}`
										: to;
								})
								.join("");
						const first = run[0]!;
						const last = run.at(-1)!;
						const d = `M${P(first.x, -0.5, first.hi)}${edge(run, "hi", 1)} L${P(last.x, 0.5, last.lo)}${edge([...run].reverse(), "lo", -1)} Z`;
						return {
							type: "path",
							shape: { pathData: d },
							style: { fill: setAlpha(colors.muted, "40") },
							// stays put while the hovered slot dims the other bars
							blur: { style: { opacity: 1 } },
						};
					},
				});
			}
			if (this.socTempValues) {
				const line = {
					id: "soctemp",
					name: "soctemp",
					type: "line",
					yAxisIndex: 1,
					data: this.socTempValues,
					smooth: true,
					symbol: "none",
					connectNulls: true,
					lineStyle: { color: colors.muted || "", ...lineDefaults },
					z: 4,
				};
				result.push(lineCasing(line, 3), line);
			}

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

			// hover dims all but the active slot (onChartMouseMove); silent stops single-segment highlight
			const barEmphasis = {
				silent: true,
				emphasis: { focus: "self" },
				blur: { itemStyle: { opacity: 0.25 } },
			};
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
			return {
				animation: true,
				animationDuration: 0,
				animationDurationUpdate: 400,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: {
					...forecastGrid(),
					left: this.socTempValues || this.socTempBands || this.prices ? 36 : 0,
					right: 36,
					...(this.showXAxis ? {} : { bottom: 4 }),
				},
				tooltip: {
					trigger: "axis",
					// transparent shadow snaps to slots without a band; triggerEmphasis off (it hard-codes notBlur), we dim slots in onChartMouseMove
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
							this.focusedEntity !== null
								? [this.focusedEntity]
								: this.series.map((s, i) => s.paletteIndex ?? i);
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

						const rows: TooltipRow[] = indices.map((i, idx) => {
							const t = rowValues[idx] ?? { energy: 0, returnEnergy: 0 };
							// a single value covers both sides, a sink can draw from above and below
							const values =
								this.isBidirectional && !this.singleValue
									? [formatValue(t.energy), formatValue(t.returnEnergy)]
									: [formatValue(t.energy + t.returnEnergy)];
							return {
								// the price row below needs a name column to line up with
								name: showName
									? (nameByIdx.get(i) ?? "")
									: this.prices
										? this.$t("energy.grid.energy")
										: undefined,
								values,
							};
						});
						const priceAt = (band?: PriceBand) => {
							const i = first.dataIndex;
							if (!band || band.avg[i] == null) return "";
							return this.period === PERIODS.DAY
								? this.fmtPricePerKWh(band.avg[i]!, this.currency)
								: this.fmtPriceRange(band.lo[i]!, band.hi[i]!, this.currency);
						};
						if (this.prices) {
							rows.push({
								name: this.$t("energy.grid.price"),
								values: [priceAt(this.prices.import), priceAt(this.prices.feedin)],
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
						// one entity with its soc or temperature: a named row per value, the
						// slot value in the day view and the bucket's range otherwise
						const fmtSocTemp = (v: number) =>
							this.socTempIsTemp ? this.fmtTemperature(v) : this.fmtPercentage(v);
						const band = this.socTempBands?.[first.dataIndex];
						const socTemp = this.socTempValues?.[first.dataIndex];
						const socTempText = band
							? band[0] === band[1]
								? fmtSocTemp(band[0])
								: `${fmtSocTemp(band[0])} – ${fmtSocTemp(band[1])}`
							: socTemp != null
								? fmtSocTemp(socTemp)
								: null;
						if (socTempText !== null && !showName) {
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
				xAxis: {
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
					axisLabel: {
						...xAxisLabelStyle(),
						show: this.showXAxis,
						hideOverlap: false,
						interval: 0,
						formatter: (_value: string, index: number) => formatLabel(cats[index] ?? 0),
					},
				},
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
					// left: soc always 0 to 100, temperature at least 30 to 70, or the
					// price range with feed-in below zero
					forecastYAxis(
						this.prices
							? {
									show: true,
									position: "left",
									// from zero unless a price goes negative
									min: (v: { min: number }) => Math.min(0, v.min),
									max: this.priceExtent,
									splitNumber: 3,
									splitLine: { show: false },
									name: this.pricePerKWhUnit(this.currency, true),
									...axisNameStyle("right"),
									axisLabel: {
										color: colors.muted || "",
										hideOverlap: true,
										formatter: (v: number) =>
											this.fmtPricePerKWh(v, this.currency, true, false),
									},
								}
							: {
									show: !!this.socTempValues || !!this.socTempBands,
									position: "left",
									min: this.socTempIsTemp
										? (v: { min: number }) => Math.min(30, v.min)
										: 0,
									max: this.socTempIsTemp
										? (v: { max: number }) => Math.max(70, v.max)
										: 100,
									splitNumber: 3,
									interval: this.socTempIsTemp ? undefined : 25,
									splitLine: { show: false },
									name: this.socTempIsTemp ? "°C" : "%",
									...axisNameStyle("right"),
									axisLabel: { color: colors.muted || "", hideOverlap: true },
								}
					),
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
		},
		resize() {
			this.chart?.resize();
			this.chartWidth = this.chart?.getWidth() ?? 0;
		},
		onChartTap(x: number, y: number) {
			this.emitSlotAt(x, y);
		},
		emitSlotAt(x: number, y: number) {
			const chart = this.chart;
			if (!chart || !chart.containPixel("grid", [x, y])) return;
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
		// highlight hovered slot, dim rest. manual because built-in axis highlight hard-codes notBlur
		onChartMouseMove(e: { offsetX: number; offsetY: number }) {
			if (!this.chart) return;
			const point: [number, number] = [e.offsetX, e.offsetY];
			if (!this.chart.containPixel({ gridIndex: 0 }, point)) {
				this.clearHighlight();
				return;
			}
			const grid = this.chart.convertFromPixel({ gridIndex: 0 }, point) as number[];
			const slot = Math.round(grid[0]!);
			if (slot === this.activeSlot) return;
			// Skip empty slots, else hovering a gap would dim the whole chart.
			if (!this.slotsWithData[slot]) {
				this.clearHighlight();
				return;
			}
			this.activeSlot = slot;
			this.chart.dispatchAction({ type: "downplay" });
			this.chart.dispatchAction({ type: "highlight", dataIndex: slot });
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
