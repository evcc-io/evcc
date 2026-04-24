<template>
	<div class="mb-5">
		<div class="chart-container my-3">
			<Bar ref="chartRef" :data="chartData" :options="chartOptions" :height="300" />
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
	BarController,
	BarElement,
	Tooltip,
	type ChartData,
	type ChartOptions,
} from "chart.js";
import { Bar } from "vue-chartjs";
import colors, { lighterColor } from "@/colors";
import { commonOptions } from "../Sessions/chartConfig";
import LegendList from "../Sessions/LegendList.vue";
import type { Legend } from "../Sessions/types";
import formatter from "@/mixins/formatter";
import type { SeriesData } from "./PowerChart.vue";

ChartJS.register(CategoryScale, LinearScale, BarController, BarElement, Tooltip);

export default defineComponent({
	name: "HistoryEnergyChart",
	components: { Bar, LegendList },
	mixins: [formatter],
	props: {
		series: { type: Array as PropType<SeriesData[]>, default: () => [] },
		from: { type: Date, required: true },
		days: { type: Number, default: 14 },
	},
	computed: {
		dayDates(): Date[] {
			const result: Date[] = [];
			for (let i = 0; i < this.days; i++) {
				const d = new Date(this.from);
				d.setDate(d.getDate() + i);
				result.push(d);
			}
			return result;
		},
		labels(): string[] {
			const locale = this.$i18n?.locale;
			const fmt = new Intl.DateTimeFormat(locale, {
				weekday: "short",
				day: "numeric",
				month: "short",
			});
			return this.dayDates.map((d) => fmt.format(d));
		},
		chartData(): ChartData<"bar"> {
			const datasets: ChartData<"bar">["datasets"] = [];
			const dayKeys = this.dayDates.map(
				(d) =>
					`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`
			);

			this.series.forEach((s, i) => {
				const color = colors.palette[i % colors.palette.length];

				// index data by day
				const byDay: Record<string, { import: number; export: number }> = {};
				s.data.forEach((slot) => {
					const d = new Date(slot.start);
					const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
					byDay[key] = { import: slot.import, export: slot.export };
				});

				const importData = dayKeys.map((key) => byDay[key]?.import ?? 0);
				const exportData = dayKeys.map((key) => -(byDay[key]?.export ?? 0));
				const hasExport = exportData.some((v) => v !== 0);

				const importLabel = hasExport ? `${s.group} (import)` : s.group;

				datasets.push({
					label: importLabel,
					data: importData,
					backgroundColor: color,
					stack: `s${i}`,
				});

				if (hasExport) {
					datasets.push({
						label: `${s.group} (export)`,
						data: exportData,
						backgroundColor: lighterColor(color),
						stack: `s${i}`,
					});
				}
			});

			return { labels: this.labels, datasets };
		},
		chartOptions(): ChartOptions<"bar"> {
			return {
				...commonOptions,
				scales: {
					x: {
						border: { display: false },
						grid: { display: false },
						ticks: {
							color: colors.muted || undefined,
							maxRotation: 0,
						},
					},
					y: {
						border: { display: false },
						title: {
							display: true,
							text: "kWh",
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
							label: (ctx) => {
								const val = ctx.parsed.y ?? 0;
								const abs = Math.abs(val);
								const label = ctx.dataset.label || "";
								return `${label}: ${abs.toFixed(1)} kWh`;
							},
						},
					},
				},
			} as ChartOptions<"bar">;
		},
		legends(): Legend[] {
			const result: Legend[] = [];
			this.series.forEach((s, i) => {
				const color = colors.palette[i % colors.palette.length];
				const hasExport = s.data.some((slot) => slot.export > 0);

				const importLabel = hasExport ? `${s.group} (import)` : s.group;
				result.push({ label: importLabel, color, value: "" });
				if (hasExport) {
					result.push({
						label: `${s.group} (export)`,
						color: lighterColor(color),
						value: "",
					});
				}
			});
			return result;
		},
	},
});
</script>
