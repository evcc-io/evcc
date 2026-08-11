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
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import { GROUPS, type Session } from "./types";
import type { DeviceColors } from "@/types/evcc";

export default defineComponent({
	name: "EnergyGroupedChart",
	components: { LegendList },
	mixins: [formatter, echartsChart],
	props: {
		sessions: { type: Array as PropType<Session[]>, default: () => [] },
		groupBy: { type: String as PropType<GROUPS>, default: GROUPS.NONE },
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {}, solar: {} }) },
		deviceColors: { type: Object as PropType<DeviceColors>, default: () => ({}) },
	},
	computed: {
		chartData(): { labels: string[]; data: number[]; colors: string[] } {
			const aggregatedData: { [key: string]: number } = {};

			if (this.groupBy === GROUPS.NONE) {
				const total = this.sessions.reduce((acc, s) => acc + s.chargedEnergy, 0);
				const self = this.sessions.reduce(
					(acc, s) => acc + (s.chargedEnergy / 100) * s.solarPercentage,
					0
				);
				aggregatedData["self"] = self;
				aggregatedData["grid"] = total - self;
			} else {
				this.sessions.forEach((session) => {
					const groupKey = session[this.groupBy as "loadpoint" | "vehicle"];
					if (!aggregatedData[groupKey]) {
						aggregatedData[groupKey] = 0;
					}
					aggregatedData[groupKey] += session.chargedEnergy;
				});
			}

			// stable alphabetical order so entries don't jump between periods
			const entries = Object.entries(aggregatedData).sort((a, b) => a[0].localeCompare(b[0]));
			const labels = entries.map(([label]) =>
				this.groupBy === GROUPS.NONE ? this.$t(`sessions.group.${label}`) : label
			);
			const data = entries.map(([, value]) => value);
			const colorGroup = this.groupBy === GROUPS.NONE ? "solar" : this.groupBy;
			const entryColors = entries.map(([label]) => this.colorMappings[colorGroup][label]);

			return { labels, data, colors: entryColors };
		},
		legends() {
			const { labels, data, colors: entryColors } = this.chartData;
			const total = data.reduce((acc, curr) => acc + curr, 0);
			const maxEnergy = Math.max(...data);
			// sync energy units for label grid view
			const unit =
				maxEnergy < 1 ? POWER_UNIT.W : maxEnergy > 1e4 ? POWER_UNIT.MW : POWER_UNIT.KW;
			const fmtShare = (value: number) => this.fmtPercentage((100 / total) * value, 1);
			const fmtValue = (value: number) => this.fmtWh(value * 1e3, unit);
			const pickable = this.groupBy !== GROUPS.NONE;
			return labels.map((label, index) => {
				const dataValue = data[index] as number;
				return {
					label,
					color: entryColors[index],
					value: [fmtValue(dataValue), fmtShare(dataValue)],
					id: pickable ? label || undefined : undefined,
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
			return this.fmtWh(value * 1e3, POWER_UNIT.AUTO);
		},
	},
});
</script>
