<template>
	<div v-if="chartData.labels.length > 1" class="row">
		<div class="col-12 col-md-6 col-lg-12 col-xxl-6 mb-3">
			<div ref="chartEl" class="round-chart"></div>
		</div>
		<div class="col-12 col-md-6 col-lg-12 col-xxl-6 d-flex align-items-center">
			<LegendList :legends="legends" :device-colors="deviceColors" grid />
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import { doughnutOption } from "./echarts";
import echartsChart from "@/mixins/echartsChart";
import LegendList from "./LegendList.vue";
import formatter from "@/mixins/formatter";
import { TYPES, GROUPS, type Session } from "./types";
import { CURRENCY, type DeviceColors } from "@/types/evcc";

export default defineComponent({
	name: "CostGroupedChart",
	components: { LegendList },
	mixins: [formatter, echartsChart],
	props: {
		sessions: { type: Array as PropType<Session[]>, default: () => [] },
		groupBy: {
			type: String as PropType<Exclude<GROUPS, GROUPS.NONE>>,
			default: GROUPS.LOADPOINT,
		},
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {} }) },
		deviceColors: { type: Object as PropType<DeviceColors>, default: () => ({}) },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
		costType: { type: String as PropType<TYPES>, default: TYPES.PRICE },
	},
	computed: {
		chartData(): { labels: string[]; data: number[]; colors: string[] } {
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

			// stable alphabetical order so entries don't jump between periods
			const sortedEntries = Object.entries(aggregatedData).sort((a, b) =>
				a[0].localeCompare(b[0])
			);
			const labels = sortedEntries.map(([label]) => label);
			const data = sortedEntries.map(([, value]) => value);
			const entryColors = labels.map((label) => this.colorMappings[this.groupBy][label]);

			return { labels, data, colors: entryColors };
		},
		legends() {
			const { labels, data, colors: entryColors } = this.chartData;
			const total = data.reduce((acc, curr) => acc + curr, 0);
			const fmtShare = (value: number) => this.fmtPercentage((100 / total) * value, 1);
			return labels.map((label, index) => {
				const dataValue = data[index] as number;
				return {
					label,
					color: entryColors[index],
					value: [this.formatValue(dataValue), fmtShare(dataValue)],
					id: label || undefined,
				};
			});
		},
		chartOption(): Record<string, unknown> {
			const { labels, data, colors: entryColors } = this.chartData;
			return doughnutOption(
				labels.map((name, i) => ({ name, value: data[i]!, color: entryColors[i]! })),
				this.formatValue,
				() => this.chart
			);
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
