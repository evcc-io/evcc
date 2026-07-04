<template>
	<div class="history-chart-wrapper" :data-testid="`group-chart-${group}`">
		<div ref="chartEl" class="history-chart"></div>
	</div>
</template>

<script lang="ts">
import { defineComponent, markRaw, type PropType } from "vue";
import {
	echarts,
	FONT_FAMILY,
	forecastGrid,
	forecastYAxis,
	tooltipStyle,
	tooltipTable,
} from "../Forecast/echarts";
import colors, { resolveColors, deviceColorMap, darken, batteryColor, setAlpha } from "@/colors";
import store from "@/store";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import { PERIODS } from "../Sessions/types";
import { is12hFormat } from "@/units";
import { hasColorPicker } from "./groups";

export interface HistorySlot {
	start: string;
	end: string;
	energy: number;
	returnEnergy: number;
}

export interface HistorySeries {
	title: string;
	group: string;
	data: HistorySlot[];
	// Marks a synthetic / derived series (e.g. "other consumers"). Gets a neutral
	// color and is excluded from the source-data table.
	virtual?: boolean;
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

// Symmetric axis regardless of whether the period contains both directions.
const BIDIRECTIONAL_GROUPS: ReadonlySet<string> = new Set(["grid", "battery"]);

// Multiple entities stack into one bar; grid and meter render side-by-side.
const STACKED_GROUPS: ReadonlySet<string> = new Set(["loadpoint", "consumer", "pv", "battery"]);

// Round up to a nice number (5-tick symmetric axis: -L, -L/2, 0, L/2, L).
function niceCeil(v: number): number {
	if (v <= 0) return 0;
	const mag = Math.pow(10, Math.floor(Math.log10(v)));
	const r = v / mag;
	const n = r <= 1 ? 1 : r <= 2 ? 2 : r <= 3 ? 3 : r <= 4 ? 4 : r <= 6 ? 6 : r <= 8 ? 8 : 10;
	return n * mag;
}

export default defineComponent({
	name: "GroupChart",
	mixins: [formatter],
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
	},
	data(): {
		chart: echarts.ECharts | null;
		isMobile: boolean;
		mediaQuery: MediaQueryList | null;
		previousFocusedEntity: number | null;
		previousPeriod: PERIODS;
		previousSeriesKey: string;
		activeSlot: number | null;
	} {
		return {
			chart: null,
			isMobile: false,
			mediaQuery: null,
			previousFocusedEntity: this.focusedEntity as number | null,
			previousPeriod: this.period as PERIODS,
			previousSeriesKey: "",
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
			return STACKED_GROUPS.has(this.group);
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
				peak(this.visibleSeries, (slot) => Math.abs(slot.energy - slot.returnEnergy)),
				peak(overlay, (slot) => slot.energy)
			);
		},
		// W/Wh scale when peak below 1 kW(h). Zero data falls here too.
		useSmallUnit(): boolean {
			return this.axisPeak < 1;
		},
		// Rounded range. W mode floors at 1 kW (= 1000 W) for stable context.
		axisLimit(): number {
			const v = niceCeil(this.axisPeak);
			return this.useSmallUnit ? Math.max(v, 1) : v;
		},
		// 1-3 kW(h) band gets one decimal to avoid duplicate integer ticks.
		axisDigits(): number {
			if (this.useSmallUnit) return 0;
			return this.axisLimit > 0 && this.axisLimit <= 3 ? 1 : 0;
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
			if (BIDIRECTIONAL_GROUPS.has(this.group)) return true;
			// additional grid/battery meters can export
			if (this.group === "meter") {
				return this.series.some((s) => s.data.some((slot) => slot.returnEnergy > 0));
			}
			return false;
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
			const has = new Array(this.categoryKeys.length).fill(false);
			for (const s of this.visibleSeries) {
				for (const slot of s.data) {
					if (slot.energy <= 0 && slot.returnEnergy <= 0) continue;
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
			const radius = 4;
			const factor = this.valueFactor;

			const result: Record<string, unknown>[] = [];

			// Always render overlay slot (line series) so series structure is stable;
			// data is all-null when toggled off. Prepend so it renders BEHIND bars.
			const overlayValues: (number | null)[] = new Array(cats.length).fill(null);
			if (this.showOverlay && this.overlay.length) {
				for (const s of this.overlay) {
					for (const slot of s.data) {
						const idx = index.get(slotKey(slot.start));
						if (idx === undefined) continue;
						const v = (slot.energy - slot.returnEnergy) * factor;
						overlayValues[idx] = (overlayValues[idx] || 0) + v;
					}
				}
			}
			const overlayCol = this.overlayColor || this.color;
			result.push({
				id: "overlay",
				name: this.overlayLabel || "overlay",
				type: "line",
				data: overlayValues,
				smooth: true,
				symbol: "none",
				lineStyle: { color: overlayCol, width: 2, type: "dotted" },
				itemStyle: { color: overlayCol },
				z: 1,
			});

			// Always render import + export series per entity, even if one direction
			// is empty (null-filled). Stable series ids/structure across renders so
			// echarts can animate value transitions instead of redrawing from zero.
			// Build value arrays per entity first so we can determine, per slot,
			// which entity is the *visible* top/bottom of the stack — that one
			// gets the rounded cap even if higher-index entities are zero/null.
			const energyByEntity: (number | null)[][] = [];
			const returnEnergyByEntity: (number | null)[][] = [];
			this.series.forEach((s, i) => {
				const energyValues: (number | null)[] = new Array(cats.length).fill(null);
				const returnEnergyValues: (number | null)[] = new Array(cats.length).fill(null);
				const hidden =
					this.focusedEntity !== null && this.focusedEntity !== (s.paletteIndex ?? i);
				if (!hidden) {
					for (const slot of s.data) {
						const idx = index.get(slotKey(slot.start));
						if (idx === undefined) continue;
						if (slot.energy > 0) energyValues[idx] = slot.energy * factor;
						if (slot.returnEnergy > 0)
							returnEnergyValues[idx] = -slot.returnEnergy * factor;
					}
				}
				energyByEntity.push(energyValues);
				returnEnergyByEntity.push(returnEnergyValues);
			});
			// Per slot: index of the topmost (largest i) entity with a non-zero
			// value. -1 = no entity has data at that slot.
			const topEnergyPerSlot: number[] = new Array(cats.length).fill(-1);
			const topReturnEnergyPerSlot: number[] = new Array(cats.length).fill(-1);
			for (let i = 0; i < this.series.length; i++) {
				for (let idx = 0; idx < cats.length; idx++) {
					if ((energyByEntity[i]![idx] ?? 0) > 0) topEnergyPerSlot[idx] = i;
					if ((returnEnergyByEntity[i]![idx] ?? 0) < 0) topReturnEnergyPerSlot[idx] = i;
				}
			}

			// hover dims all but the active slot (onChartMouseMove); silent stops single-segment highlight
			const barEmphasis = {
				silent: true,
				emphasis: { focus: "self" },
				blur: { itemStyle: { opacity: 0.25 } },
			};
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
				result.push({
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

			return result;
		},
		labelForTimestamp(): (t: number) => string {
			if (this.period === PERIODS.DAY) {
				// Skip 00:00 so the chart can align with the section title on the left.
				// Use every 6h on mobile (06/12/18) and every 3h on desktop. 12 is a
				// multiple of both, so noon stays visible. Drop minutes; honor am/pm.
				const stepHours = this.isMobile ? 6 : 3;
				const h12 = is12hFormat();
				return (t: number) => {
					const d = new Date(t);
					if (d.getMinutes() !== 0) return "";
					const h = d.getHours();
					if (h === 0) return "";
					if (h % stepHours !== 0) return "";
					if (h12) {
						const hh = h % 12 || 12;
						return `${hh} ${h < 12 ? "AM" : "PM"}`;
					}
					return String(h);
				};
			}
			if (this.period === PERIODS.MONTH) {
				return (t: number) => `${new Date(t).getDate()}`;
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
			const energyKey = `main.history.direction.${this.group}.energy`;
			const returnEnergyKey = `main.history.direction.${this.group}.returnEnergy`;
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
				grid: { ...forecastGrid(), left: 0, right: 36 },
				tooltip: {
					trigger: "axis",
					// transparent shadow snaps to slots without a band; triggerEmphasis off (it hard-codes notBlur), we dim slots in onChartMouseMove
					axisPointer: {
						type: "shadow",
						triggerEmphasis: false,
						shadowStyle: { color: "transparent" },
					},
					...tooltipStyle(this.tooltipColor),
					// Allow the tooltip to float above the 180px chart container instead
					// of being clamped by `confine: true` — otherwise tall bars push the
					// tooltip onto the bar.
					confine: false,
					position: (
						point: [number, number],
						params:
							| { value: number | null; seriesId: string }[]
							| { value: number | null; seriesId: string },
						el: HTMLElement
					): [number, number] => {
						const w = el?.offsetWidth || 0;
						const h = el?.offsetHeight || 0;
						const margin = 8;
						// Anchor the tooltip just above the top edge of the bar at this
						// slot. Top edge = sum of positive imports; for export-only slots
						// (bidirectional groups with discharge) that's 0 (the zero line),
						// which still sits above the visible bar.
						const arr = Array.isArray(params) ? params : [params];
						let sum = 0;
						let hasBar = false;
						for (const p of arr) {
							if (!/^entity-\d+-(energy|returnEnergy)$/.test(p.seriesId || ""))
								continue;
							if (p.value == null) continue;
							hasBar = true;
							if (typeof p.value === "number" && p.value > 0) {
								if (/-energy$/.test(p.seriesId)) sum += p.value;
							}
						}
						let x = point[0] - w / 2;
						let y = point[1] - h - margin;
						if (hasBar && this.chart) {
							const pixelY = this.chart.convertToPixel({ yAxisIndex: 0 }, sum);
							if (typeof pixelY === "number" && isFinite(pixelY)) {
								y = pixelY - h - margin;
							}
						}
						// Clamp X to the viewport so the tooltip never escapes the browser
						// edges. The chart container is in CSS-pixel coordinates relative
						// to the chart's bounding box, so map via getBoundingClientRect.
						const dom = this.chart?.getDom();
						const rect = dom?.getBoundingClientRect();
						if (rect) {
							const minX = -rect.left + margin;
							const maxX = window.innerWidth - rect.left - w - margin;
							if (x < minX) x = minX;
							if (x > maxX) x = maxX;
						}
						return [x, y];
					},
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
						const formatValue = (v: number) => {
							const watts = Math.abs(v) * 1000;
							return this.period === PERIODS.DAY
								? this.fmtW(watts, POWER_UNIT.AUTO)
								: this.fmtWh(watts, POWER_UNIT.AUTO);
						};

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

						const rows = indices.map((i) => {
							const t = totals.get(i) ?? { energy: 0, returnEnergy: 0 };
							const values = this.isBidirectional
								? [formatValue(t.energy), formatValue(t.returnEnergy)]
								: [formatValue(t.energy + t.returnEnergy)];
							return {
								name: showName ? (nameByIdx.get(i) ?? "") : undefined,
								values,
							};
						});
						return tooltipTable(head, rows, this.directionHeaders ?? undefined);
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
						color: colors.muted || "",
						fontSize: 11,
						hideOverlap:
							this.period !== PERIODS.DAY &&
							!(this.period === PERIODS.YEAR && this.isMobile),
						interval:
							this.period === PERIODS.DAY ||
							(this.period === PERIODS.YEAR && this.isMobile)
								? 0
								: "auto",
						formatter: (_value: string, index: number) => formatLabel(cats[index] ?? 0),
					},
				},
				yAxis: forecastYAxis({
					...(this.isBidirectional && this.axisLimit > 0
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
					nameLocation: "end",
					nameGap: 18,
					nameTextStyle: {
						color: colors.muted || "",
						fontFamily: FONT_FAMILY,
						fontSize: 10,
						opacity: 0.75,
						align: "left",
						// Axis name anchors at the axis line; axis labels have a default
						// 8px margin, so shift the name right by the same amount to land
						// flush with the value labels' left edge.
						padding: [0, 0, 0, 8],
					},
					axisLabel: {
						color: colors.muted || "",
						hideOverlap: true,
						formatter: (v: number): string => {
							const unit = this.useSmallUnit ? POWER_UNIT.W : POWER_UNIT.KW;
							return this.period === PERIODS.DAY
								? this.fmtW(v * 1000, unit, false, this.axisDigits)
								: this.fmtWh(v * 1000, unit, false, this.axisDigits);
						},
					},
				}),
				series: this.echartsSeries,
			};
		},
	},
	watch: {
		chartOption: {
			handler() {
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
				const fullReset = periodChanged || newSeriesKey !== this.previousSeriesKey;
				this.chart?.setOption(
					fullReset
						? opt
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
			},
			deep: true,
		},
	},
	mounted() {
		const el = this.$refs["chartEl"] as HTMLElement;
		this.chart = markRaw(echarts.init(el));
		this.chart.setOption((this as unknown as WithChartOption).chartOption);
		const zr = this.chart.getZr();
		zr.on("mousemove", this.onChartMouseMove);
		zr.on("globalout", this.clearHighlight);
		this.mediaQuery = window.matchMedia("(max-width: 575.98px)");
		this.isMobile = this.mediaQuery.matches;
		this.mediaQuery.addEventListener("change", this.onMediaChange);
		window.addEventListener("resize", this.resize);
	},
	beforeUnmount() {
		window.removeEventListener("resize", this.resize);
		this.mediaQuery?.removeEventListener("change", this.onMediaChange);
		this.chart?.dispose();
	},
	methods: {
		resize() {
			this.chart?.resize();
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
		niceCeil(v: number): number {
			if (v <= 0) return 0;
			const mag = Math.pow(10, Math.floor(Math.log10(v)));
			const r = v / mag;
			let n;
			if (r <= 1) n = 1;
			else if (r <= 2) n = 2;
			else if (r <= 2.5) n = 2.5;
			else if (r <= 5) n = 5;
			else n = 10;
			return n * mag;
		},
		directionLabel(s: HistorySeries, dir: "energy" | "returnEnergy"): string {
			const key = `main.history.direction.${s.group}.${dir}`;
			const label = this.$t(key);
			if (label === key) return s.title;
			if (this.series.length > 1) return `${s.title} ${label}`;
			return String(label);
		},
		singleEntityName(s: HistorySeries): string {
			if (this.series.length > 1) return s.title;
			const key = `main.history.group.${s.group}`;
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
	height: 180px;
}
</style>
