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
import { defineComponent, type PropType } from "vue";
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
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import colors from "@/colors";
import { GROUPS, type Session } from "./types";

registerChartComponents([DoughnutController, ArcElement, LinearScale, Legend, Tooltip]);

export default defineComponent({
	name: "EnergyGroupedChart",
	components: { Doughnut, LegendList },
	mixins: [formatter],
	props: {
		sessions: { type: Array as PropType<Session[]>, default: () => [] },
		groupBy: { type: String as PropType<GROUPS>, default: GROUPS.NONE },
		colorMappings: { type: Object, default: () => ({ loadpoint: {}, vehicle: {}, solar: {} }) },
	},
	computed: {
		chartData() {
			console.log("update energy aggregate data");
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

			// Sort the data by energy in descending order
			const sortedEntries = Object.entries(aggregatedData); //.sort((a, b) => b[1] - a[1]);

			const labels = sortedEntries.map(([label]) =>
				this.groupBy === GROUPS.NONE ? this.$t(`sessions.group.${label}`) : label
			);
			const data = sortedEntries.map(([, value]) => value);
			const colorGroup = this.groupBy === GROUPS.NONE ? "solar" : this.groupBy;
			const backgroundColor = sortedEntries.map(
				([label]) => this.colorMappings[colorGroup][label]
			);

			return {
				labels: labels,
				datasets: [{ data, backgroundColor }],
			};
		},
		legends() {
			const total = this.chartData.datasets[0].data.reduce((acc, curr) => acc + curr, 0);
			const maxEnergy = Math.max(...this.chartData.datasets[0].data);
			// sync energy units for label grid view
			const unit =
				maxEnergy < 1 ? POWER_UNIT.W : maxEnergy > 1e4 ? POWER_UNIT.MW : POWER_UNIT.KW;
			const fmtShare = (value: number) => this.fmtPercentage((100 / total) * value, 1);
			const fmtValue = (value: number) => this.fmtWh(value * 1e3, unit);
			return this.chartData.labels.map((label, index) => ({
				label: label,
				color: this.chartData.datasets[0].backgroundColor[index],
				value: [
					fmtValue(this.chartData.datasets[0].data[index]),
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
				color: colors.text,
				borderWidth: 3,
				borderColor: colors.background,
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
								this.formatValue(tooltipItem.raw as number),
							labelColor: tooltipLabelColor(false),
						},
					},
				},
			} as any;
		},
	},
	methods: {
		formatValue(value: number) {
			return this.fmtWh(value * 1e3, POWER_UNIT.AUTO);
		},
	},
});
</script>
