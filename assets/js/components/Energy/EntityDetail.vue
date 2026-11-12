<template>
	<div>
		<CardHeader
			class="mb-3"
			:title="title"
			:parent="parent"
			:small="subhead"
			@parent="$emit('close')"
		>
			<template v-if="$slots['icon']" #icon><slot name="icon" /></template>
			<template v-if="pickable" #dot>
				<DeviceColorDot :title="title" :color="color" :explicit="deviceColor" />
			</template>
			<template #stats>
				<StatRow :stats="shownStats" inline />
			</template>
			<template #ring>
				<SourcesButton :rows="sources" @click="showSources" />
			</template>
			<template v-if="chartToggle" #actions>
				<EntityChartToggle :chart="chart" @update:chart="$emit('update:chart', $event)" />
			</template>
		</CardHeader>
		<PatternChart
			v-if="chart === ENTITY_CHART.PATTERN && daily"
			:series="daily"
			:color="color"
			:from="from"
			:to="to"
			:period="period"
			:height="chartHeight"
		/>
		<!-- the pattern's data is on its way: hold the place, bars would flash -->
		<div
			v-else-if="chart === ENTITY_CHART.PATTERN"
			:style="{ height: `${chartHeight}px` }"
		></div>
		<GroupChart
			v-else-if="series"
			:group="series.group"
			:color="color"
			:series="[{ ...series, color }]"
			:period="period"
			:from="from"
			:to="to"
			:height="chartHeight"
			:show-x-axis="showXAxis"
			soc-temp
			@slot="$emit('drill', $event)"
		/>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import StatRow from "./StatRow.vue";
import SourcesButton from "./SourcesButton.vue";
import CardHeader from "../Helper/CardHeader.vue";
import GroupChart, { type HistorySeries } from "./GroupChart.vue";
import PatternChart from "./PatternChart.vue";
import EntityChartToggle from "./EntityChartToggle.vue";
import formatter from "@/mixins/formatter";
import { CURRENCY } from "@/types/evcc";
import { PERIODS } from "../Sessions/types";
import { ENTITY_CHART, type Flow, type FlowSink, type FlowSource, type StatItem } from "./types";
import DeviceColorDot from "../Helper/DeviceColorDot.vue";
import { sinkPrice, sourceBreakdown, sumEnergy, type SourceRow } from "./mix";

// one entity of a sink group (consumer of home, loadpoint): bars or pattern chart, its
// energy, cost and source mix in the header. The crumb closes
export default defineComponent({
	name: "EntityDetail",
	components: {
		StatRow,
		SourcesButton,
		GroupChart,
		PatternChart,
		EntityChartToggle,
		DeviceColorDot,
		CardHeader,
	},
	mixins: [formatter],
	props: {
		title: { type: String, required: true },
		color: { type: String, default: "" },
		sink: { type: String as PropType<FlowSink>, default: "home" },
		// the group this entity was picked from, a crumb before the title that closes
		parent: { type: String, default: "" },
		// bars only, no pattern chart
		chartToggle: { type: Boolean, default: true },
		// the title as a smaller subhead inside a titled card
		subhead: Boolean,
		chartHeight: { type: Number, default: 200 },
		showXAxis: { type: Boolean, default: true },
		// the dot behind the title opens the color picker for the device
		pickable: Boolean,
		// numbers for the header instead of energy and cost
		stats: Array as PropType<StatItem[]>,
		deviceColor: { type: String, default: "" }, // configured color, empty when automatic
		// what reached this entity over the period, priced per slot
		flows: { type: Array as PropType<Flow[]>, default: () => [] },
		// sources the site has, others never show up in the breakdown
		flowSources: { type: Array as PropType<FlowSource[]>, default: () => [] },
		series: { type: Object as PropType<HistorySeries> },
		period: { type: String as PropType<PERIODS>, required: true },
		from: { type: Date, required: true },
		to: { type: Date, required: true },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
		daily: { type: Object as PropType<HistorySeries> },
		chart: { type: String as PropType<ENTITY_CHART>, default: ENTITY_CHART.BARS },
	},
	emits: ["close", "update:chart", "drill", "sources", "sources-change"],
	data() {
		return { ENTITY_CHART };
	},
	watch: {
		// an open breakdown follows the period
		sources: {
			handler() {
				this.$emit("sources-change", this.sourcesDetail());
			},
			immediate: true,
		},
	},
	methods: {
		sourcesDetail() {
			return { owner: "consumer", title: this.title, rows: this.sources };
		},
		showSources() {
			this.$emit("sources", this.sourcesDetail());
		},
	},
	computed: {
		shownStats(): StatItem[] {
			if (this.stats) return this.stats;
			const list: StatItem[] = [
				{
					key: "energy",
					label: this.$t("energy.stat.energy"),
					number: this.energy,
					format: (v: number) => this.fmtKWh(v),
				},
			];
			if (this.cost !== undefined) {
				list.push({
					key: "cost",
					label: this.$t("energy.consumers.cost"),
					number: this.cost,
					format: (v: number) => this.fmtMoneyWithSymbol(v, this.currency),
					sub: `ø ${this.fmtPricePerKWh(this.price, this.currency)}`,
				});
			}
			return list;
		},
		// the entity's source split, scaled to its metered energy
		sources(): SourceRow[] {
			return sourceBreakdown(this.flows, this.sink, this.energy, this.flowSources);
		},
		energy(): number {
			return this.series ? sumEnergy(this.series) : 0;
		},
		// per kWh over the slots with a price, undefined without tariffs
		price(): number | undefined {
			return sinkPrice(this.flows, this.sink);
		},
		cost(): number | undefined {
			return this.price === undefined ? undefined : this.energy * this.price;
		},
	},
});
</script>
