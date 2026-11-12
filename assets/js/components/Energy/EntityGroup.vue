<template>
	<Card
		:title="active ?? only?.title ?? title"
		:parent="active || only ? title : ''"
		:parent-static="!!only"
		edge-to-edge
		class="box-pull-out mb-4"
		data-anchor
		@parent="focused = null"
	>
		<template v-if="$slots['icon']" #icon><slot name="icon" /></template>
		<template v-if="!isBattery && shown.length === 1" #dot>
			<DeviceColorDot
				:title="shown[0]?.title"
				:color="leadColor"
				:explicit="deviceColors[shown[0]?.title ?? ''] || ''"
			/>
		</template>
		<template #actions>
			<EntityChartToggle :chart="chart" @update:chart="$emit('update:chart', $event)" />
		</template>
		<template #stats>
			<StatRow :stats="stats" inline :selected="selectedStat" @select="selectDirection" />
		</template>
		<template #ring>
			<SourcesButton :rows="sources" @click="showSources" />
		</template>
		<PatternChart
			v-if="pattern"
			:series="pattern"
			:value-label="patternLabel"
			:color="leadColor"
			:from="from"
			:to="to"
			:period="period"
			:height="200"
		/>
		<!-- the pattern's data is on its way: hold the place, bars would flash -->
		<div v-else-if="wantsPattern" class="pattern-pending"></div>
		<GroupChart
			v-else
			:group="group"
			:color="leadColor"
			:series="series"
			:focused-entity="focusedIndex"
			:soc-temp="shown.length === 1"
			:period="period"
			:from="from"
			:to="to"
			:height="200"
			@slot="$emit('drill', $event)"
		/>
		<!-- a single entity is the header, a legend would repeat it -->
		<LegendList
			v-if="!only"
			class="mt-4"
			:legends="legends"
			:device-colors="deviceColors"
			columns
			@focus="toggle"
		/>
	</Card>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Card from "../Helper/Card.vue";
import StatRow from "./StatRow.vue";
import SourcesButton from "./SourcesButton.vue";
import GroupChart, { type HistorySeries } from "./GroupChart.vue";
import PatternChart from "./PatternChart.vue";
import LegendList from "../Sessions/LegendList.vue";
import type { Legend } from "../Sessions/types";
import EntityChartToggle from "./EntityChartToggle.vue";
import DeviceColorDot from "../Helper/DeviceColorDot.vue";
import formatter from "@/mixins/formatter";
import { CURRENCY, type DeviceColors } from "@/types/evcc";
import { PERIODS } from "../Sessions/types";
import { ENTITY_CHART, type EntityFlows, type FlowSource, type StatItem } from "./types";
import {
	entityFlows,
	mergeSources,
	sinkPrice,
	sourceBreakdown,
	sumEnergy,
	sumSeries,
	type SourceRow,
} from "./mix";

// the entities of one group (chargers, heaters, batteries) in one card: stacked bars
// with the sums in the header. A legend row focuses its entity, chart and numbers
// then show it alone. A single entity is the group, there is nothing to focus
export default defineComponent({
	name: "EntityGroup",
	components: {
		Card,
		StatRow,
		SourcesButton,
		GroupChart,
		PatternChart,
		LegendList,
		EntityChartToggle,
		DeviceColorDot,
	},
	mixins: [formatter],
	props: {
		title: { type: String, required: true },
		group: { type: String as PropType<"loadpoint" | "battery">, required: true },
		entities: { type: Array as PropType<HistorySeries[]>, default: () => [] },
		colors: { type: Object as PropType<Record<string, string>>, default: () => ({}) },
		deviceColors: { type: Object as PropType<DeviceColors>, default: () => ({}) },
		// what reached each entity over the period, priced per slot
		flows: { type: Array as PropType<EntityFlows[]>, default: () => [] },
		// sources the site has, others never show up in the breakdown
		flowSources: { type: Array as PropType<FlowSource[]>, default: () => [] },
		// finer buckets of the entities for the pattern chart
		daily: { type: Array as PropType<HistorySeries[]>, default: () => [] },
		period: { type: String as PropType<PERIODS>, required: true },
		from: { type: Date, required: true },
		to: { type: Date, required: true },
		currency: { type: String as PropType<CURRENCY>, default: CURRENCY.EUR },
		chart: { type: String as PropType<ENTITY_CHART>, default: ENTITY_CHART.BARS },
		// what the pattern of a battery shows, it has one value per cell
		direction: { type: String as PropType<"charged" | "discharged">, default: "charged" },
	},
	emits: ["update:chart", "update:direction", "drill", "sources", "sources-change"],
	data() {
		return { focused: null as string | null };
	},
	computed: {
		isBattery(): boolean {
			return this.group === "battery";
		},
		// kept across periods, applies only while the entity is one of several
		active(): string | null {
			const known = this.entities.some((s) => s.title === this.focused);
			return this.entities.length > 1 && known ? this.focused : null;
		},
		// the group's one entity, named in the title behind the group as a crumb
		only(): HistorySeries | undefined {
			return this.entities.length === 1 ? this.entities[0] : undefined;
		},
		shown(): HistorySeries[] {
			return this.active
				? this.entities.filter((s) => s.title === this.active)
				: this.entities;
		},
		series(): HistorySeries[] {
			return this.entities.map((s, i) => ({
				...s,
				color: this.colors[s.title] || "",
				paletteIndex: i,
			}));
		},
		focusedIndex(): number | null {
			return this.active ? this.entities.findIndex((s) => s.title === this.active) : null;
		},
		leadColor(): string {
			return this.colors[this.shown[0]?.title ?? ""] || "";
		},
		// bars or the pattern of what is shown
		wantsPattern(): boolean {
			return this.chart === ENTITY_CHART.PATTERN;
		},
		patternLabel(): string | undefined {
			return this.isBattery ? this.$t(`energy.battery.${this.direction}`) : undefined;
		},
		// a battery's pattern shows one direction, picked by its tile in the header
		selectedStat(): string | undefined {
			return this.isBattery && this.wantsPattern ? this.direction : undefined;
		},
		pattern(): HistorySeries | null {
			if (!this.wantsPattern) return null;
			const titles = this.shown.map((s) => s.title);
			const list = this.daily.filter((s) => titles.includes(s.title));
			if (!list.length) return null;
			const data = sumSeries(list);
			const discharged = this.isBattery && this.direction === "discharged";
			return {
				title: this.title,
				group: this.group,
				data: discharged
					? data.map((slot) => ({ ...slot, energy: slot.returnEnergy }))
					: data,
			};
		},
		energy(): number {
			return this.shown.reduce((acc, s) => acc + sumEnergy(s), 0);
		},
		// each entity at its own average price, undefined without priced slots
		cost(): number | undefined {
			const costs = this.shown
				.map((s) => this.entityCost(s))
				.filter((c): c is number => c !== undefined);
			return costs.length ? costs.reduce((acc, c) => acc + c, 0) : undefined;
		},
		sources(): SourceRow[] {
			if (this.isBattery) return [];
			return mergeSources(
				this.shown.map((s) =>
					sourceBreakdown(
						entityFlows(this.flows, "loadpoint", s.title),
						"loadpoint",
						sumEnergy(s),
						this.flowSources
					)
				)
			);
		},
		stats(): StatItem[] {
			const kWh = (v: number) => this.fmtKWh(v);
			if (this.isBattery) {
				const discharged = this.shown.reduce(
					(acc, s) => acc + sumEnergy(s, "returnEnergy"),
					0
				);
				return [
					{
						key: "charged",
						label: this.$t("energy.battery.charged"),
						number: this.energy,
						format: kWh,
					},
					{
						key: "discharged",
						label: this.$t("energy.battery.discharged"),
						number: discharged,
						format: kWh,
					},
				];
			}
			const list: StatItem[] = [
				{
					key: "energy",
					label: this.$t("energy.stat.energy"),
					number: this.energy,
					format: kWh,
				},
			];
			if (this.cost !== undefined) {
				list.push({
					key: "cost",
					label: this.$t("energy.consumers.cost"),
					number: this.cost,
					format: (v: number) => this.fmtMoneyWithSymbol(v, this.currency),
					sub: this.energy
						? `ø ${this.fmtPricePerKWh(this.cost / this.energy, this.currency)}`
						: "",
				});
			}
			return list;
		},
		legends(): Legend[] {
			return this.entities.map((s) => {
				const cost = this.entityCost(s);
				const second = this.isBattery
					? this.fmtKWh(sumEnergy(s, "returnEnergy"))
					: cost === undefined
						? undefined
						: this.fmtMoneyWithSymbol(cost, this.currency);
				return {
					label: s.title,
					color: this.colors[s.title] || "",
					value: second ? [this.fmtKWh(sumEnergy(s)), second] : this.fmtKWh(sumEnergy(s)),
					// batteries keep their palette color
					id: this.isBattery ? undefined : s.title,
					focusable: true,
					focusKey: s.title,
					dim: !!this.active && this.active !== s.title,
				};
			});
		},
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
		entityCost(s: HistorySeries): number | undefined {
			if (this.isBattery) return undefined;
			const price = sinkPrice(entityFlows(this.flows, "loadpoint", s.title), "loadpoint");
			return price === undefined ? undefined : sumEnergy(s) * price;
		},
		selectDirection(key: string) {
			this.$emit("update:direction", key);
		},
		// a second click on the focused entity shows the group again
		toggle(legend: Legend) {
			const key = legend.focusKey as string;
			this.focused = this.active === key ? null : key;
		},
		sourcesDetail() {
			// one entity under its own name, several under the group's
			const [only, ...more] = this.shown;
			const title = only && !more.length ? only.title : this.title;
			return { owner: this.title, title, rows: this.sources };
		},
		showSources() {
			this.$emit("sources", this.sourcesDetail());
		},
	},
});
</script>

<style scoped>
/* as high as the charts */
.pattern-pending {
	height: 200px;
}
</style>
