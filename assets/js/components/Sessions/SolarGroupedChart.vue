<template>
	<div v-if="chartData.labels.length > 1" class="row">
		<div class="col-12 col-md-6 mb-3">
			<PolarArea :data="chartData" :options="options" />
		</div>
		<div class="col-12 col-md-6 d-flex align-items-center py-0 py-md-3">
			<LegendList :legends="legends" grid />
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { PolarArea } from "vue-chartjs";
import { RadialLinearScale, ArcElement, Legend, Tooltip, type TooltipItem } from "chart.js";
import { registerChartComponents, commonOptions, tooltipLabelColor } from "./chartConfig.ts";
import formatter from "@/mixins/formatter";
import colors, { dimColor } from "@/colors";
import LegendList from "./LegendList.vue";
import { GROUPS, type Session } from "./types.ts";

registerChartComponents([RadialLinearScale, ArcElement, Legend, Tooltip]);

export default defineComponent({
	name: "SolarGroupedChart",
	components: { PolarArea, LegendList },
	mixins: [formatter],
	props: {
		sessions: { type: Array as PropType<Session[]>, default: () => [] },
		groupBy: {
			type: String as PropType<Exclude<GROUPS, GROUPS.NONE>>,
			default: GROUPS.LOADPOINT,
		},
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
	},
	computed: {
		chartData() {
			console.log("update solar grouped data");
			const aggregatedData: Record<string, { grid: number; self: number }> = {};

			this.sessions.forEach((session) => {
				const groupKey = session[this.groupBy];
				if (!aggregatedData[groupKey]) {
					aggregatedData[groupKey] = { grid: 0, self: 0 };
				}
				const charged = session.chargedEnergy;
				const self = (charged / 100) * session.solarPercentage;
				const grid = charged - self;
				aggregatedData[groupKey].self += self;
				aggregatedData[groupKey].grid += grid;
			});

			// Sort the data by total energy in descending order
			const sortedEntries = Object.entries(aggregatedData).sort(
				(a, b) => b[1].grid + b[1].self - (a[1].grid + a[1].self)
			);
			const labels = sortedEntries.map(([label]) => label);
			const data = sortedEntries.map(([, value]) => {
				const total = value.grid + value.self;
				const selfPercentage = (value.self / total) * 100;
				return selfPercentage;
			});

			const borderColors = labels.map((label) => this.colorMappings[this.groupBy][label]);
			const backgroundColors = borderColors.map((color) => dimColor(color));
			return {
				labels: labels,
				datasets: [
					{
						data: data,
						borderColor: borderColors,
						backgroundColor: backgroundColors,
					},
				],
			};
		},
		legends() {
			return this.chartData.labels.map((label, index) => ({
				label: label,
				color: this.chartData.datasets[0].borderColor[index],
				value: this.fmtPercentage(this.chartData.datasets[0].data[index], 1),
			}));
		},
		options() {
			return {
				...commonOptions,
				locale: this.$i18n?.locale,
				aspectRatio: 1,
				borderRadius: 8,
				borderWidth: 3,
				color: colors.text,
				spacing: 0,
				radius: "100%",
				plugins: {
					...commonOptions.plugins,
					tooltip: {
						...commonOptions.plugins.tooltip,
						axis: "r",
						position: "topBottomCenter",
						callbacks: {
							title: () => null,
							label: (tooltipItem: TooltipItem<"polarArea">) => {
								const { label, dataset, dataIndex } = tooltipItem;
								const d = dataset.data[dataIndex];

								return label + ": " + this.fmtPercentage(d, 1);
							},
							labelColor: tooltipLabelColor(true),
						},
					},
				},
				scales: {
					r: {
						min: 0,
						max: 100,
						ticks: {
							stepSize: 25,
							color: colors.muted,
							backdropColor: colors.background,
						},
						grid: { color: colors.border },
					},
				},
			} as any;
		},
	},
});
</script>
