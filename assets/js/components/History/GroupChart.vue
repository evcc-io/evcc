<template>
	<div class="history-chart-wrapper">
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
} from "../Forecast/echarts";
import colors, { lighterColor } from "@/colors";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import { PERIODS } from "../Sessions/types";
import { is12hFormat } from "@/units";
import { groupExportColor } from "./groups";

export interface HistorySlot {
	start: string;
	end: string;
	import: number;
	export: number;
}

export interface HistorySeries {
	name: string;
	group: string;
	data: HistorySlot[];
	// Marks a synthetic / derived series (e.g. "other consumers"). Gets a neutral
	// color and is excluded from the source-data table.
	virtual?: boolean;
}

type WithChartOption = { chartOption: Record<string, unknown> };

// Alpha (0..1) for entity at position `i` in a `n`-entity stack: top is opaque,
// each step below fades by 20%, capped at 50%.
export function stepAlpha(i: number, n: number): number {
	const step = 0.2;
	const minAlpha = 0.5;
	return Math.max(minAlpha, 1 - (n - 1 - i) * step);
}

export function alphaColor(color: string, alpha: number): string {
	const c = (color || "").trim().toLowerCase();
	const a = Math.round(Math.max(0, Math.min(1, alpha)) * 255)
		.toString(16)
		.padStart(2, "0");
	if (c.length === 7) return c + a;
	if (c.length === 9) return c.slice(0, 7) + a;
	return c;
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
	} {
		return {
			chart: null,
			isMobile: false,
			mediaQuery: null,
			previousFocusedEntity: this.focusedEntity as number | null,
			previousPeriod: this.period as PERIODS,
		};
	},
	computed: {
		// In day view a slot is 15 minutes of energy (kWh). Display as
		// average power (kW): 15 min slot ⇒ kW = kWh × 4.
		valueFactor(): number {
			return this.period === PERIODS.DAY ? 4 : 1;
		},
		unit(): "kW" | "kWh" {
			return this.period === PERIODS.DAY ? "kW" : "kWh";
		},
		// Consumer groups have many palette colours per entity — a single neutral
		// tooltip background reads better than picking one entity's colour.
		tooltipColor(): string {
			if (this.group === "loadpoint" || this.group === "meter") {
				return colors.text || this.color;
			}
			return this.color;
		},
		visibleSeries(): HistorySeries[] {
			if (this.focusedEntity === null) return this.series;
			const idx = this.focusedEntity;
			return this.series.filter((_, i) => i === idx);
		},
		isBidirectional(): boolean {
			for (const s of this.visibleSeries) {
				let imp = 0;
				let exp = 0;
				for (const slot of s.data) {
					imp += slot.import;
					exp += slot.export;
				}
				if (imp > 0 && exp > 0) return true;
			}
			return false;
		},
		// Soft symmetric limit. Rounded up to an even-friendly nice number so
		// splitNumber: 4 gives integer ticks ±X, ±X/2, 0.
		bidirectionalLimit(): number {
			let m = 0;
			const factor = this.valueFactor;
			for (const s of this.visibleSeries) {
				for (const slot of s.data) {
					const a = Math.abs(slot.import * factor);
					const b = Math.abs(slot.export * factor);
					if (a > m) m = a;
					if (b > m) m = b;
				}
			}
			if (m <= 0) return 2;
			const mag = Math.pow(10, Math.floor(Math.log10(m)));
			const r = m / mag;
			let n;
			if (r <= 2) n = 2;
			else if (r <= 4) n = 4;
			else if (r <= 6) n = 6;
			else if (r <= 8) n = 8;
			else n = 10;
			return n * mag;
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
		entryColors(): string[] {
			// Loadpoint and meter use the palette per entity (distinct entities).
			// Production and battery use the group color with subtle alpha steps so
			// stacked segments stay visually distinguishable.
			if (this.group === "loadpoint" || this.group === "meter") {
				const mutedColor = colors.muted || this.color;
				return this.series.map((s, i) =>
					// Virtual "other consumers" entity renders in a neutral gray to set
					// it apart from explicit meter entities.
					s.virtual ? mutedColor : colors.palette[i % colors.palette.length] || this.color
				);
			}
			if (this.series.length <= 1) return [this.color];
			return this.series.map((_, i) =>
				alphaColor(this.color, stepAlpha(i, this.series.length))
			);
		},
		echartsSeries() {
			const cats = this.categoryKeys;
			const index = new Map<string, number>();
			cats.forEach((k, i) => index.set(k, i));
			const slotKey = (start: string) => this.timestampKey(new Date(start).getTime());
			const radius = this.period === PERIODS.DAY ? 2 : 6;
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
						const v = (slot.import - slot.export) * factor;
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
			// Groups with multiple entities stack them; grid keeps bars side-by-side.
			const stackEntities =
				this.group === "loadpoint" ||
				this.group === "meter" ||
				this.group === "pv" ||
				this.group === "battery";
			const lastIdx = this.series.length - 1;
			this.series.forEach((s, i) => {
				const c = this.entryColors[i] || this.color;
				const exportColor = groupExportColor(s.group) || lighterColor(c) || c;
				const importValues: (number | null)[] = new Array(cats.length).fill(null);
				const exportValues: (number | null)[] = new Array(cats.length).fill(null);
				const hidden = this.focusedEntity !== null && this.focusedEntity !== i;
				if (!hidden) {
					for (const slot of s.data) {
						const idx = index.get(slotKey(slot.start));
						if (idx === undefined) continue;
						if (slot.import > 0) importValues[idx] = slot.import * factor;
						if (slot.export > 0) exportValues[idx] = -slot.export * factor;
					}
				}
				const importName =
					this.series.length > 1 || this.isBidirectional
						? this.directionLabel(s, "import")
						: this.singleEntityName(s);
				const exportName = this.directionLabel(s, "export");
				// Same stack name for import and export means they share one x slot
				// (positive values stack up, negative stack down, no width penalty).
				const stackName = stackEntities ? `group-${this.group}` : `entity-${i}`;
				const importStack = stackName;
				const exportStack = stackName;
				// In stacked consumer groups only the topmost entity caps the bar;
				// other segments stay flat so we get one rounded top per column.
				const importRadius =
					stackEntities && i !== lastIdx ? [0, 0, 0, 0] : [radius, radius, 0, 0];
				const exportRadius = [0, 0, radius, radius];
				result.push({
					id: `entity-${i}-import`,
					name: importName,
					type: "bar",
					stack: importStack,
					data: importValues,
					itemStyle: { color: c, borderRadius: importRadius },
					barCategoryGap: "25%",
					barGap: "10%",
				});
				result.push({
					id: `entity-${i}-export`,
					name: exportName,
					type: "bar",
					stack: exportStack,
					data: exportValues,
					itemStyle: { color: exportColor, borderRadius: exportRadius },
					barCategoryGap: "25%",
					barGap: "10%",
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
					axisPointer: { type: "shadow", shadowStyle: { color: "transparent" } },
					...tooltipStyle(this.tooltipColor),
					position: (
						point: [number, number],
						_params: unknown,
						el: HTMLElement
					): [number, number] => {
						const w = el?.offsetWidth || 0;
						const h = el?.offsetHeight || 0;
						return [point[0] - w / 2, point[1] - h - 12];
					},
					formatter: (
						params: { value: number | null; seriesName: string; dataIndex: number }[]
					) => {
						if (!params?.length) return "";
						const visible = params.filter((p) => p.value != null);
						if (!visible.length) return "";
						const first = visible[0];
						if (!first) return "";
						const ts = cats[first.dataIndex];
						const head = `<div>${ts != null ? tooltipDate(ts) : ""}</div>`;
						const rows = visible
							.map((p) => {
								const watts = Math.abs(p.value || 0) * 1000;
								const val =
									this.period === PERIODS.DAY
										? this.fmtW(watts, POWER_UNIT.KW, true, 1)
										: this.fmtWh(watts, POWER_UNIT.KW, true, 1);
								const showName = visible.length > 1;
								if (showName) {
									return `<div>${p.seriesName}: <strong>${val}</strong></div>`;
								}
								return `<div><strong>${val}</strong></div>`;
							})
							.join("");
						return head + rows;
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
					...(this.isBidirectional
						? {
								min: -this.bidirectionalLimit,
								max: this.bidirectionalLimit,
								interval: this.bidirectionalLimit / 2,
							}
						: {}),
					position: "right",
					splitNumber: 3,
					splitLine: {
						showMinLine: true,
						showMaxLine: true,
						lineStyle: { color: colors.border || "" },
					},
					axisLabel: {
						color: colors.muted || "",
						hideOverlap: true,
						formatter: (v: number) =>
							this.period === PERIODS.DAY
								? this.fmtW(v * 1000, POWER_UNIT.KW, false, 0)
								: this.fmtWh(v * 1000, POWER_UNIT.KW, false, 0),
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
				// Snap (no animation) on period switches and on legend focus toggles.
				// Other updates use the default merge so bars animate value transitions
				// per stable key.
				this.chart?.setOption({
					animation: !(focusChanged || periodChanged),
					xAxis: opt["xAxis"],
					yAxis: opt["yAxis"],
					series: opt["series"],
				});
				this.previousFocusedEntity = this.focusedEntity as number | null;
				this.previousPeriod = this.period as PERIODS;
			},
			deep: true,
		},
	},
	mounted() {
		const el = this.$refs["chartEl"] as HTMLElement;
		this.chart = markRaw(echarts.init(el));
		this.chart.setOption((this as unknown as WithChartOption).chartOption);
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
		directionLabel(s: HistorySeries, dir: "import" | "export"): string {
			const key = `main.history.direction.${s.group}.${dir}`;
			const label = this.$t(key);
			if (label === key) return s.name;
			if (this.series.length > 1) return `${s.name} ${label}`;
			return String(label);
		},
		singleEntityName(s: HistorySeries): string {
			if (this.series.length > 1) return s.name;
			const key = `main.history.group.${s.group}`;
			const label = this.$t(key);
			return label === key ? s.name : String(label);
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
