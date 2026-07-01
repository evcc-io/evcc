<template>
	<div ref="chartEl" class="battery-chart"></div>
</template>

<script lang="ts">
import { defineComponent, markRaw, type PropType } from "vue";
import {
	echarts,
	FONT_FAMILY,
	forecastYAxis,
	tooltipStyle,
	tooltipTable,
	type TooltipRow,
} from "../Forecast/echarts";
import colors, { dimColor, lighterColor, batteryColor } from "@/colors";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import { is12hFormat } from "@/units";
import type { SocPoint, BatterySeries } from "./types";

type EChartsType = ReturnType<typeof echarts.init>;
type Point = [number, number];

export default defineComponent({
	name: "BatteryHistoryChart",
	mixins: [formatter],
	props: {
		batteries: { type: Array as PropType<BatterySeries[]>, default: () => [] },
		mode: { type: String as PropType<"soc" | "energy">, default: "soc" },
		winStart: { type: Number, required: true },
		winEnd: { type: Number, required: true },
		now: { type: Number, required: true },
		hasForecast: Boolean,
		dayOffset: { type: Number, default: 0 }, // paging position; slide animates only when it changes
		focused: { type: Number as PropType<number | null>, default: null },
	},
	data() {
		return {
			chart: null as EChartsType | null,
			previousMode: "soc" as string,
			prevWinStart: 0,
			prevWinEnd: 0,
			prevDayOffset: 0,
			hasRendered: false,
			slideRaf: undefined as number | undefined,
		};
	},
	computed: {
		single(): boolean {
			return this.batteries.length === 1;
		},
		nowInWindow(): boolean {
			return this.now >= this.winStart && this.now <= this.winEnd;
		},
		totalCapacity(): number {
			return this.batteries.reduce((s, b) => s + b.capacity, 0);
		},
		totalCurrentKWh(): number {
			return this.batteries.reduce((s, b) => s + (b.currentSoc / 100) * b.capacity, 0);
		},
		yMax(): number {
			if (this.mode === "soc") return 100;
			return Math.max(Math.ceil(this.totalCapacity / 5) * 5, 5);
		},
		energyGrid(): number[] {
			// union of all timestamps (full preloaded range) for stack alignment; the
			// x-axis min/max clips to the visible window, so paging only shifts the axis
			const set = new Set<number>([this.now]);
			for (const b of this.batteries) {
				for (const p of [...b.history, ...b.forecast]) set.add(p.t);
			}
			return [...set].sort((a, b) => a - b);
		},
		echartsSeries(): Record<string, unknown>[] {
			const series: Record<string, unknown>[] = [];
			if (this.mode === "soc") {
				this.batteries.forEach((b, i) => {
					if (this.focused !== null && this.focused !== i) return;
					const c = batteryColor(i);
					series.push({
						id: this.seriesId(b.id, "hist"),
						type: "line",
						z: 3,
						data: this.socHistory(b),
						showSymbol: false,
						lineStyle: { color: c, width: 2 },
						itemStyle: { color: c },
						...(this.single
							? {
									areaStyle: {
										color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
											{ offset: 0, color: lighterColor(c) || c },
											{ offset: 1, color: (c || "#000000") + "00" },
										]),
									},
								}
							: {}),
						emphasis: { disabled: true },
					});
					series.push({
						id: this.seriesId(b.id, "fc"),
						type: "line",
						z: 3,
						data: this.socForecast(b),
						showSymbol: false,
						lineStyle: { color: c, width: 2, type: "dashed" },
						itemStyle: { color: c },
						...(this.single ? { areaStyle: { color: dimColor(c) } } : {}),
						emphasis: { disabled: true },
					});
				});
			} else {
				this.batteries.forEach((b, i) => {
					if (this.focused !== null && this.focused !== i) return;
					const c = batteryColor(i);
					series.push({
						id: this.seriesId(b.id, "hist"),
						type: "line",
						stack: "e-hist",
						z: 2 + i,
						data: this.energyData(b, "hist"),
						showSymbol: false,
						smooth: 0.4,
						lineStyle: { width: 0 },
						itemStyle: { color: c },
						areaStyle: { color: c },
						emphasis: { disabled: true },
					});
					series.push({
						id: this.seriesId(b.id, "fc"),
						type: "line",
						stack: "e-fc",
						z: 2 + i,
						data: this.energyData(b, "fc"),
						showSymbol: false,
						smooth: 0.4,
						lineStyle: { color: c, width: 1, type: "dashed" },
						itemStyle: { color: c },
						areaStyle: { color: dimColor(c) },
						emphasis: { disabled: true },
					});
				});
			}
			// now marker as a 2-point series so it slides with the axis on paging
			if (this.nowInWindow) {
				series.push({
					id: "now-line",
					type: "line",
					data: [
						[this.now, 0],
						[this.now, this.yMax],
					],
					showSymbol: false,
					silent: true,
					z: 5,
					lineStyle: { color: colors.muted || "", width: 1, type: "dashed" },
					emphasis: { disabled: true },
				});
			}
			// overlay carries the forecast-region shading
			series.push({
				id: "overlay",
				type: "line",
				data: [],
				silent: true,
				z: 4,
				markArea: this.markArea,
			});
			return series;
		},
		markArea(): Record<string, unknown> {
			const data =
				this.hasForecast && this.nowInWindow
					? [[{ xAxis: this.now }, { xAxis: this.winEnd }]]
					: [];
			return { silent: true, itemStyle: { color: colors.muted || "", opacity: 0.06 }, data };
		},
		chartOption(): Record<string, unknown> {
			return {
				// no echarts animation; paging is driven manually via slideWindow
				animation: false,
				textStyle: { fontFamily: FONT_FAMILY },
				grid: { top: 10, right: 36, bottom: 26, left: 0, borderWidth: 0 },
				tooltip: {
					trigger: "axis",
					axisPointer: {
						type: "line",
						lineStyle: { color: colors.muted || "", opacity: 0.4 },
					},
					...tooltipStyle(colors.text || "", () => this.chart),
					formatter: this.tooltipFormatter,
				},
				xAxis: this.xAxes,
				yAxis: forecastYAxis({
					min: 0,
					max: this.yMax,
					position: "right",
					...(this.mode === "soc" ? { interval: 25 } : { splitNumber: 5 }),
					axisLabel: {
						color: colors.muted || "",
						formatter: (v: number) => this.fmtNumber(v, 0),
					},
				}),
				series: this.echartsSeries,
			};
		},
		xAxes(): Record<string, unknown>[] {
			const h12 = is12hFormat();
			return [
				// hours every 6h, midnight handled by the day axis below
				{
					type: "time",
					min: this.winStart,
					max: this.winEnd,
					minInterval: 6 * 3600 * 1000,
					axisLine: { show: false },
					axisTick: { show: false },
					splitLine: { show: false },
					axisLabel: {
						color: colors.muted || "",
						fontSize: 11,
						formatter: (value: number) => {
							const d = new Date(value);
							const h = d.getHours();
							if (d.getMinutes() !== 0 || h === 0 || h % 6 !== 0) return "";
							return h12 ? `${h % 12 || 12} ${h < 12 ? "AM" : "PM"}` : String(h);
						},
					},
				},
				// day axis: a tick at every local midnight, labelled with the weekday + dashed divider
				{
					type: "time",
					position: "bottom",
					min: this.winStart,
					max: this.winEnd,
					minInterval: 24 * 3600 * 1000,
					maxInterval: 24 * 3600 * 1000,
					axisLine: { show: false },
					axisTick: { show: false },
					axisLabel: {
						color: colors.muted || "",
						fontSize: 11,
						formatter: (value: number) => this.weekdayShort(new Date(value)),
					},
					splitLine: {
						show: true,
						showMinLine: false,
						showMaxLine: false,
						lineStyle: { color: colors.border || "", type: "dashed" },
					},
				},
			];
		},
	},
	watch: {
		chartOption: {
			handler() {
				const modeChanged = this.previousMode !== this.mode;
				const windowChanged =
					this.prevWinStart !== this.winStart || this.prevWinEnd !== this.winEnd;

				// a running slide re-renders from the latest chartOption every frame, so data that
				// loads mid-slide is picked up automatically. only intervene if the target changed.
				if (this.slideRaf && !windowChanged && !modeChanged) return;

				// animate only on real paging (dayOffset change); forecast arrival and clock
				// advance also shift the window but must not trigger the slide
				const paging =
					this.hasRendered && !modeChanged && this.dayOffset !== this.prevDayOffset;
				const fromStart = this.prevWinStart;
				const fromEnd = this.prevWinEnd;

				this.cancelSlide();
				this.previousMode = this.mode;
				this.prevWinStart = this.winStart;
				this.prevWinEnd = this.winEnd;
				this.prevDayOffset = this.dayOffset;

				if (paging) {
					this.slideWindow(fromStart, fromEnd, this.winStart, this.winEnd);
				} else {
					this.chart?.setOption(
						{ ...this.chartOption, animation: false },
						modeChanged ? { notMerge: true } : { replaceMerge: ["series"] }
					);
					this.hasRendered = true;
					this.$nextTick(() => this.updateGraphic());
				}
			},
			deep: true,
		},
	},
	mounted() {
		const el = this.$refs["chartEl"] as HTMLElement;
		this.chart = markRaw(echarts.init(el));
		this.chart.setOption(this.chartOption);
		this.$nextTick(() => this.updateGraphic());
		window.addEventListener("resize", this.resize);
		// initial render done here (not via the watcher), so paging animates from the first click
		this.hasRendered = true;
		this.prevWinStart = this.winStart;
		this.prevWinEnd = this.winEnd;
		this.prevDayOffset = this.dayOffset;
		this.previousMode = this.mode;
	},
	beforeUnmount() {
		window.removeEventListener("resize", this.resize);
		this.cancelSlide();
		this.chart?.dispose();
	},
	methods: {
		// stable echarts series id per battery/part; single source for building and tooltip lookup
		seriesId(id: string, part: "hist" | "fc"): string {
			return `${this.mode}-${id}-${part}`;
		},
		resize() {
			this.chart?.resize();
			this.updateGraphic();
		},
		cancelSlide() {
			if (this.slideRaf) cancelAnimationFrame(this.slideRaf);
			this.slideRaf = undefined;
		},
		// tween the x-axis window ourselves so axis and series slide together (echarts cannot
		// animate axis min/max, so changing them directly snaps the labels/gridlines). each frame
		// re-renders from the latest chartOption, so data loaded mid-slide is picked up seamlessly.
		slideWindow(fromStart: number, fromEnd: number, toStart: number, toEnd: number) {
			const dur = 450;
			const ease = (p: number) => 1 - Math.pow(1 - p, 3); // cubicOut
			const apply = (min: number, max: number) => {
				this.chart?.setOption(
					{
						...this.chartOption,
						animation: false,
						xAxis: [
							{ min, max },
							{ min, max },
						],
					},
					{ replaceMerge: ["series"] }
				);
				this.updateGraphic();
			};
			apply(fromStart, fromEnd); // start position, synchronous, before first paint
			this.hasRendered = true;
			const t0 = performance.now();
			const step = (t: number) => {
				const p = Math.min(1, (t - t0) / dur);
				const k = ease(p);
				apply(fromStart + (toStart - fromStart) * k, fromEnd + (toEnd - fromEnd) * k);
				this.slideRaf = p < 1 ? requestAnimationFrame(step) : undefined;
			};
			this.slideRaf = requestAnimationFrame(step);
		},
		interp(points: SocPoint[], t: number): number | null {
			if (!points.length) return null;
			if (t <= points[0]!.t) return points[0]!.soc;
			if (t >= points[points.length - 1]!.t) return points[points.length - 1]!.soc;
			for (let i = 1; i < points.length; i++) {
				const a = points[i - 1]!;
				const b = points[i]!;
				if (t <= b.t) {
					const f = (t - a.t) / (b.t - a.t);
					return a.soc + f * (b.soc - a.soc);
				}
			}
			return points[points.length - 1]!.soc;
		},
		socHistory(b: BatterySeries): Point[] {
			// full history up to now; the x-axis min/max clips to the window
			const pts: Point[] = b.history.filter((p) => p.t <= this.now).map((p) => [p.t, p.soc]);
			pts.push([this.now, b.currentSoc]);
			return pts;
		},
		socForecast(b: BatterySeries): Point[] {
			if (!b.forecast.length) return [];
			const pts: Point[] = [[this.now, b.currentSoc]];
			for (const p of b.forecast) {
				if (p.t > this.now) pts.push([p.t, p.soc]);
			}
			return pts.length > 1 ? pts : [];
		},
		energyData(b: BatterySeries, part: "hist" | "fc"): Point[] {
			const all = [...b.history, ...b.forecast].sort((a, c) => a.t - c.t);
			const factor = b.capacity / 100;
			const first = all[0]?.t ?? Infinity;
			const last = all.at(-1)?.t ?? -Infinity;
			return this.energyGrid
				.filter((t) => (part === "hist" ? t <= this.now : t >= this.now))
				.map((t) => {
					// contribute nothing outside this battery's known range instead of
					// back-/forward-extending its edge soc across the stacked band
					if (t < first || t > last) return [t, 0] as Point;
					const soc = this.interp(all, t);
					return [t, soc == null ? 0 : soc * factor] as Point;
				});
		},
		tooltipFormatter(
			params: { axisValue: number; seriesId?: string; value?: Point }[]
		): string {
			const arr = Array.isArray(params) ? params : [params];
			if (!arr.length) return "";
			const t = new Date(arr[0]!.axisValue);
			const head = `${this.weekdayShort(t)} ${this.fmtHourMinute(t)}`;
			const byId = new Map<string, number>();
			for (const p of arr) {
				const v = p.value?.[1];
				if (v != null && !Number.isNaN(v) && p.seriesId) byId.set(p.seriesId, v);
			}
			const rows: TooltipRow[] = [];
			this.batteries.forEach((b) => {
				// past points come from the hist series, future ones from the fc series
				const v =
					byId.get(this.seriesId(b.id, "hist")) ?? byId.get(this.seriesId(b.id, "fc"));
				if (v == null) return;
				const val =
					this.mode === "soc"
						? this.fmtPercentage(v)
						: this.fmtWh(v * 1e3, POWER_UNIT.KW, true, 1);
				rows.push({ name: this.batteries.length > 1 ? b.title : undefined, values: [val] });
			});
			if (!rows.length) return "";
			return tooltipTable(head, rows);
		},
		updateGraphic() {
			if (!this.chart) return;
			const finder = { xAxisIndex: 0, yAxisIndex: 0 };
			const elements: Record<string, unknown>[] = [];
			if (this.nowInWindow) {
				// badge sits to the left of the now point (matches the design, avoids clipping the right edge)
				const badge = (px: number[], text: string, fill: string, textFill = "#fff") => {
					elements.push({
						type: "circle",
						z: 10,
						shape: { cx: px[0], cy: px[1], r: 4 },
						style: { fill, stroke: colors.background || "", lineWidth: 2 },
					});
					elements.push({
						type: "text",
						z: 11,
						x: px[0] - 9,
						y: px[1],
						style: {
							text,
							fill: textFill,
							font: `bold 12px ${FONT_FAMILY}`,
							align: "right",
							verticalAlign: "middle",
							backgroundColor: fill,
							padding: [4, 9],
							borderRadius: 4,
						},
					});
				};
				if (this.mode === "soc") {
					this.batteries.forEach((b, i) => {
						if (this.focused !== null && this.focused !== i) return;
						const px = this.chart!.convertToPixel(finder, [this.now, b.currentSoc]);
						if (px) badge(px, this.fmtPercentage(b.currentSoc), batteryColor(i));
					});
				} else {
					// focused: that battery's energy (its band is the only one shown); else the total
					const f = this.focused;
					const kWh =
						f !== null && this.batteries[f]
							? (this.batteries[f]!.currentSoc / 100) * this.batteries[f]!.capacity
							: this.totalCurrentKWh;
					const px = this.chart.convertToPixel(finder, [this.now, kWh]);
					if (px) {
						const label = this.fmtWh(kWh * 1e3, POWER_UNIT.KW, true, 1);
						if (f !== null) {
							badge(px, label, batteryColor(f));
						} else {
							// group total sits on the stack top, so use a neutral (not battery) color
							badge(px, label, colors.text || "", colors.background || "#fff");
						}
					}
				}
			}
			this.chart.setOption({ graphic: { elements } }, { replaceMerge: ["graphic"] });
		},
	},
});
</script>

<style scoped>
.battery-chart {
	width: 100%;
	height: 260px;
}
@media (max-width: 575.98px) {
	.battery-chart {
		height: 220px;
	}
}
</style>
