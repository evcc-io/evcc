<template>
	<div v-if="chartData.labels.length > 1" class="row">
		<div class="col-12 col-md-6 mb-3">
			<Doughnut :data="chartData" :options="options" />
		</div>
		<div class="col-12 col-md-6 d-flex align-items-center">
			<LegendList :legends="legends" grid />
		</div>
	</div>
</template>

<script lang="ts">
import { Doughnut } from "vue-chartjs";
import {
	DoughnutController,
	ArcElement,
	LinearScale,
	Legend,
	Tooltip,
	type TooltipItem,
} from "chart.js";
import LegendList from "./LegendList.vue";
import { registerChartComponents, commonOptions, tooltipLabelColor } from "./chartConfig";
import formatter from "@/mixins/formatter";
import colors from "@/colors";
import { TYPES, GROUPS, type Session } from "./types";
import { defineComponent, type PropType } from "vue";
import { CURRENCY } from "@/types/evcc";

registerChartComponents([DoughnutController, ArcElement, LinearScale, Legend, Tooltip]);

export default defineComponent({
	name: "CostGroupedChart",
	components: { Doughnut, LegendList },
	mixins: [formatter],
	props: {
		sessions: { type: Array as PropType<Session[]>, default: () => [] },
		groupBy: {
			type: String as PropType<Exclude<GROUPS, GROUPS.NONE>>,
			default: GROUPS.LOADPOINT,
		},
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
		costType: { type: String as PropType<TYPES>, default: TYPES.PRICE },
	},
	computed: {
		chartData() {
			console.log(`update ${this.costType} grouped data`);
			const aggregatedData: Record<string, number> = {};

			this.sessions.forEach((session) => {
				const groupKey = session[this.groupBy];
				if (!aggregatedData[groupKey]) {
					aggregatedData[groupKey] = 0;
				}
				if (this.costType === TYPES.PRICE) {
					aggregatedData[groupKey] += session.price || 0;
				} else if (this.costType === TYPES.CO2) {
					aggregatedData[groupKey] +=
						(session.co2PerKWh || 0) * (session.chargedEnergy || 0);
				}
			});

			const sortedEntries = Object.entries(aggregatedData).sort((a, b) => b[1] - a[1]);
			const labels = sortedEntries.map(([label]) => label);
			const data = sortedEntries.map(([, value]) => value);
			const backgroundColor = labels.map((label) => this.colorMappings[this.groupBy][label]);

			return {
				labels: labels,
				datasets: [{ data, backgroundColor }],
			};
		},
		legends() {
			const total = this.chartData.datasets[0].data.reduce((acc, curr) => acc + curr, 0);
			const fmtShare = (value: number) => this.fmtPercentage((100 / total) * value, 1);
			return this.chartData.labels.map((label, index) => ({
				label: label,
				color: this.chartData.datasets[0].backgroundColor[index],
				value: [
					this.formatValue(this.chartData.datasets[0].data[index]),
					fmtShare(this.chartData.datasets[0].data[index]),
				],
			}));
		},
		options() {
			return {
				...commonOptions,
				locale: this.$i18n?.locale,
				aspectRatio: 1,
				borderRadius: 10,
				color: colors.text || "",
				borderWidth: 3,
				borderColor: colors.background || "",
				cutout: "70%",
				radius: "95%",
				animation: { duration: 250 },
				plugins: {
					...commonOptions.plugins,
					tooltip: {
						...commonOptions.plugins.tooltip,
						axis: "r",
						position: "center",
						callbacks: {
							label: (tooltipItem: TooltipItem<"doughnut">) =>
								this.formatValue(
									tooltipItem.dataset.data[tooltipItem.dataIndex] || 0
								),
							labelColor: tooltipLabelColor(false),
						},
					},
				},
			} as any;
		},
	},
	methods: {
		formatValue(value: number) {
			if (this.costType === TYPES.PRICE) {
				return this.fmtMoney(value, this.currency, true, true);
			}
			return this.fmtGrams(value);
		},
	},
});
</script>
