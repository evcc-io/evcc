<template>
	<div ref="chartEl" class="river" data-testid="consumer-river"></div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { FONT_FAMILY, tooltipStyle, tooltipTable, xAxisLabelStyle } from "./echarts";
import echartsChart from "@/mixins/echartsChart";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import colors, { setAlpha } from "@/colors";
import { PERIODS } from "../Sessions/types";
import type { HistorySeries } from "./GroupChart.vue";
import { SLOT_MS, slotWatts } from "./slots";
import { labelStep, DAY_STEPS, MONTH_STEPS } from "@/utils/labelStep";
import { is12hFormat } from "@/units";

const NARROW_WIDTH = 576;
// the bar charts share their row with a stats column on large screens, the stream
// spans the card. Labels step as if it were as wide as they are
const LARGE_SCREEN = 992;
const STATS_WIDTH = 224;

export interface RiverLayer {
	key: string; // consumer title or OTHERS
	name: string;
	color: string;
	series: HistorySeries;
}

// the consumers over the period as a stream: one flat layer each, as thick as its
// energy at that time
export default defineComponent({
	name: "ConsumerRiver",
	mixins: [formatter, echartsChart],
	props: {
		layers: { type: Array as PropType<RiverLayer[]>, default: () => [] },
		period: { type: String as PropType<PERIODS>, required: true },
		from: { type: Date, required: true },
		to: { type: Date, required: true },
		selected: { type: String as PropType<string | null>, default: null },
	},
	emits: ["select"],
	data() {
		return { chartWidth: 0 };
	},
	computed: {
		// labels at the same steps as the bar charts: every other hour or day on a
		// wide chart, thinning out evenly on narrow ones
		labelWidth(): number {
			const stats = window.innerWidth >= LARGE_SCREEN ? STATS_WIDTH : 0;
			return this.chartWidth - stats;
		},
		ticks(): number[] {
			const [y, m, d] = [this.from.getFullYear(), this.from.getMonth(), this.from.getDate()];
			const every = (count: number, step: number, at: (i: number) => Date) =>
				Array.from({ length: Math.ceil(count / step) }, (_, i) => at(i * step).getTime());
			switch (this.period) {
				case PERIODS.DAY: {
					const width = is12hFormat() ? 56 : 40;
					const step = labelStep(24, this.labelWidth, DAY_STEPS, width);
					// 00:00 skipped like the bar charts
					return every(24, step, (h) => new Date(y, m, d, h)).slice(1);
				}
				case PERIODS.MONTH: {
					const days = new Date(y, m + 1, 0).getDate();
					const step = labelStep(31, this.labelWidth, MONTH_STEPS);
					return every(days, step, (i) => new Date(y, m, i + 1));
				}
				default:
					return every(12, 1, (i) => new Date(y, i, 1));
			}
		},
		// the stream stacks downwards, drawn in reverse the first layer is at the bottom
		stack(): RiverLayer[] {
			return [...this.layers].reverse();
		},
		// energy per layer and bucket start
		buckets(): Map<number, number>[] {
			return this.stack.map(
				(layer) =>
					new Map(
						layer.series.data.map((slot) => [
							new Date(slot.start).getTime(),
							slot.energy,
						])
					)
			);
		},
		// every layer needs a value at every time, missing buckets count as zero
		data(): [number, number, string][] {
			const times = [...new Set(this.buckets.flatMap((values) => [...values.keys()]))].sort(
				(a, b) => a - b
			);
			return this.stack.flatMap((layer, i) =>
				times.map((t): [number, number, string] => [
					t,
					this.buckets[i]!.get(t) ?? 0,
					layer.name,
				])
			);
		},
		chartOption(): Record<string, unknown> {
			return {
				animation: false,
				textStyle: { fontFamily: FONT_FAMILY },
				// the others fade while one consumer is selected, like the treemap's tiles
				color: this.stack.map((layer) =>
					this.selected && this.selected !== layer.key
						? setAlpha(layer.color, "59") || layer.color
						: layer.color
				),
				tooltip: {
					trigger: "axis",
					axisPointer: {
						type: "line",
						lineStyle: { color: colors.muted || "", opacity: 0.4 },
						label: { show: false },
					},
					...tooltipStyle(colors.text || ""),
					formatter: (params: { value: [number, number, string] }[]) => {
						if (!params?.length) return "";
						// only what ran at that time, the list is long otherwise
						const rows = [...params]
							.reverse()
							.filter((p) => p.value[1] > 0)
							.map((p) => ({
								name: p.value[2],
								values: [this.fmtValue(p.value[1])],
							}));
						const total = params.reduce((acc, p) => acc + p.value[1], 0);
						return tooltipTable(this.fmtTime(params[0]!.value[0]), [
							...rows,
							{
								name: this.$t("sessions.total"),
								values: [this.fmtValue(total)],
								total: true,
							},
						]);
					},
				},
				singleAxis: {
					type: "time",
					min: this.from.getTime(),
					max: this.to.getTime(),
					top: 8,
					bottom: 24,
					left: 0,
					right: 0,
					axisLine: { show: false },
					axisTick: { show: false },
					splitLine: { show: false },
					axisLabel: {
						...xAxisLabelStyle(),
						customValues: this.ticks,
						// the period's first label sits on the left edge
						alignMinLabel: "left",
						formatter: (v: number) => this.fmtTick(v),
					},
				},
				series: [
					{
						type: "themeRiver",
						data: this.data,
						boundaryGap: [0, 0],
						label: { show: false },
						// flat layers, hovering changes nothing, a click still selects
						emphasis: { disabled: true },
					},
				],
			};
		},
	},
	methods: {
		onChartResize() {
			this.chartWidth = this.chart?.getWidth() ?? 0;
		},
		onChartInit() {
			this.onChartResize();
			// a layer's event carries its index in the stack, not a data item
			this.chart?.on("click", (params) => {
				const layer = this.stack[params.dataIndex];
				if (layer) this.$emit("select", layer.key);
			});
		},
		// 15 minute slot as average power, longer buckets as energy
		fmtValue(energy: number): string {
			return this.period === PERIODS.DAY
				? this.fmtW(slotWatts(energy), POWER_UNIT.AUTO)
				: this.fmtKWh(energy);
		},
		fmtTime(t: number): string {
			const d = new Date(t);
			switch (this.period) {
				case PERIODS.DAY:
					return this.fmtTimeSlot(d, SLOT_MS);
				case PERIODS.MONTH:
					return this.fmtDayMonth(d);
				default:
					return this.fmtMonthYear(d);
			}
		},
		fmtTick(t: number): string {
			const d = new Date(t);
			switch (this.period) {
				case PERIODS.DAY:
					return this.hourShort(d);
				case PERIODS.MONTH:
					return `${d.getDate()}`;
				default:
					// single letters on phones, like the bar charts
					return this.chartWidth < NARROW_WIDTH
						? this.fmtMonthNarrow(d)
						: this.fmtMonth(d, true);
			}
		},
	},
});
</script>

<style scoped>
@import "../../../css/breakpoints.css";

/* same frame as the treemap, toggling does not jump */
.river {
	height: 360px;
}
@media (--sm-and-up) {
	.river {
		height: 260px;
	}
}
</style>
