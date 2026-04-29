<template>
	<div class="mb-5">
		<div class="chart-container my-3">
			<Line ref="chartRef" :data="chartData" :options="chartOptions" :height="300" />
		</div>
		<LegendList :legends="legends" />
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import {
	Chart as ChartJS,
	LinearScale,
	TimeScale,
	LineController,
	LineElement,
	PointElement,
	Filler,
	Tooltip,
	type ChartData,
	type ChartOptions,
} from "chart.js";
import "chartjs-adapter-dayjs-4/dist/chartjs-adapter-dayjs-4.esm";
import { Line } from "vue-chartjs";
import colors, { dimColor } from "@/colors";
import { commonOptions } from "../Sessions/chartConfig";
import LegendList from "../Sessions/LegendList.vue";
import type { Legend } from "../Sessions/types";
import formatter from "@/mixins/formatter";
import { is12hFormat } from "@/units";

ChartJS.register(
	LinearScale,
	TimeScale,
	LineController,
	LineElement,
	PointElement,
	Filler,
	Tooltip
);

interface Slot {
	start: string;
	end: string;
	import: number;
	export: number;
}

export interface SeriesData {
	name?: string;
	group: string;
	data: Slot[];
}

function slotPower(slot: Slot, field: "import" | "export"): number {
	const hours = (new Date(slot.end).getTime() - new Date(slot.start).getTime()) / 3_600_000;
	return hours > 0 ? slot[field] / hours : 0;
}

export default defineComponent({
	name: "HistoryPowerChart",
	components: { Line, LegendList },
	mixins: [formatter],
	props: {
		series: { type: Array as PropType<SeriesData[]>, default: () => [] },
		from: { type: Date, required: true },
		to: { type: Date, required: true },
	},
	computed: {
		chartData(): ChartData<"line"> {
			const datasets = this.series.map((s, i) => {
				const color = colors.palette[i % colors.palette.length];
				const fill = dimColor(color);

				const points = s.data.map((slot) => {
					const start = new Date(slot.start).getTime();
					const end = new Date(slot.end).getTime();
					return {
						x: (start + end) / 2,
						y: slotPower(slot, "import") - slotPower(slot, "export"),
					};
				});

				return {
					label: s.group,
					data: points,
					borderColor: color,
					backgroundColor: fill,
					fill: true,
					pointRadius: 0,
					borderWidth: 1.5,
					tension: 0.05,
				};
			});

			return { datasets } as ChartData<"line">;
		},
		chartOptions(): ChartOptions<"line"> {
			const locale = this.$i18n?.locale;
			const fmtTime = new Intl.DateTimeFormat(locale, {
				hour: "2-digit",
				minute: "2-digit",
				hour12: is12hFormat(),
			});
			const fmtDayShort = new Intl.DateTimeFormat(locale, {
				weekday: "short",
				day: "numeric",
			});
			return {
				...commonOptions,
				scales: {
					x: {
						type: "time",
						min: this.from.getTime(),
						max: this.to.getTime(),
						border: { display: false },
						grid: { display: false },
						ticks: {
							maxTicksLimit: 12,
							color: colors.muted || undefined,
							autoSkip: true,
							maxRotation: 0,
							callback: (value: number | string) => {
								const d = new Date(value);
								if (d.getHours() === 0 && d.getMinutes() === 0) {
									return [fmtTime.format(d), fmtDayShort.format(d)];
								}
								return fmtTime.format(d);
							},
						},
					},
					y: {
						border: { display: false },
						suggestedMin: 0,
						title: {
							display: true,
							text: "kW",
							color: colors.muted || undefined,
						},
						ticks: {
							color: colors.muted || undefined,
							callback: (value: number | string) => Number(value).toFixed(1),
						},
						grid: {
							color: (ctx: { tick: { value: number } }) =>
								ctx.tick.value === 0
									? colors.muted || undefined
									: colors.border || undefined,
						},
					},
				},
				plugins: {
					...commonOptions.plugins,
					tooltip: {
						...commonOptions.plugins.tooltip,
						callbacks: {
							title: (items) => {
								const val = items[0]?.parsed?.x ?? 0;
								if (!val) return "";
								const d = new Date(val);
								return `${fmtDayShort.format(d)} ${fmtTime.format(d)}`;
							},
							label: (ctx) => {
								const val = (ctx.parsed.y ?? 0).toFixed(1);
								return `${ctx.dataset.label}: ${val} kW`;
							},
						},
					},
				},
			} as ChartOptions<"line">;
		},
		legends(): Legend[] {
			return this.series.map((s, i) => ({
				label: s.group,
				color: colors.palette[i % colors.palette.length],
				value: "",
			}));
		},
	},
});
</script>
