<template>
	<div>
		<CardHeader class="header" :title="title" :subtitle="subtitle" :small="subhead">
			<template v-if="pickable" #prefix>
				<DeviceColorDot :title="title" :color="color" :explicit="deviceColor" />
			</template>
			<template v-if="$slots['icon']" #icon><slot name="icon" /></template>
			<template v-if="chartToggle || closable" #actions>
				<IconSelectGroup v-if="chartToggle">
					<IconSelectItem
						v-for="option in chartOptions"
						:key="option.value"
						:active="chart === option.value"
						:title="option.name"
						@click="$emit('update:chart', option.value)"
					>
						<component :is="option.icon" />
					</IconSelectItem>
				</IconSelectGroup>
				<button
					v-if="closable"
					type="button"
					class="btn-close"
					:aria-label="$t('config.general.close')"
					@click="$emit('close')"
				></button>
			</template>
		</CardHeader>
		<div class="d-lg-flex gap-5">
			<div class="chart-col flex-grow-1">
				<PatternChart
					v-if="chart === ENTITY_CHART.PATTERN && daily"
					:series="daily"
					:color="color"
					:from="from"
					:to="to"
					:period="period"
					:height="chartHeight"
				/>
				<GroupChart
					v-else-if="series"
					:group="series.group"
					:color="color"
					:series="[{ ...series, color }]"
					:period="period"
					:from="from"
					:to="to"
					:height="chartHeight"
					:show-x-axis="showXAxis || !isLarge"
					soc-temp
					@slot="$emit('drill', $event)"
				/>
			</div>
			<div
				class="aside row gy-2 gy-lg-4 gx-sm-5 mt-3 mt-lg-0 pt-lg-2 flex-lg-column justify-content-lg-center flex-shrink-0"
			>
				<slot name="aside">
					<div v-if="cost !== undefined && price !== undefined" class="col-6 col-lg-12">
						<Stat
							:label="$t('energy.consumers.cost')"
							:number="cost"
							:format="(v: number) => fmtMoneyWithSymbol(v, currency)"
							:sub="`ø ${fmtPricePerKWh(price, currency)}`"
							compact
						/>
					</div>
					<div v-if="hasOwnSources" class="col-6 col-lg-12">
						<button
							type="button"
							class="btn p-0 border-0 d-block w-100"
							@click="showSources"
						>
							<Stat
								:label="$t('energy.sources.title')"
								:align="sourcesAlign"
								:number="mainSource.share"
								:format="fmtMainSource"
								compact
							>
								<template #sub>
									<span class="text-decoration-underline">{{ sourceText }}</span>
								</template>
							</Stat>
						</button>
					</div>
				</slot>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Stat from "./Stat.vue";
import CardHeader from "../Helper/CardHeader.vue";
import GroupChart, { type HistorySeries } from "./GroupChart.vue";
import PatternChart from "./PatternChart.vue";
import IconSelectGroup from "../Helper/IconSelectGroup.vue";
import IconSelectItem from "../Helper/IconSelectItem.vue";
import BarChartIcon from "../MaterialIcon/BarChart.vue";
import FullStackedBarChartIcon from "../MaterialIcon/FullStackedBarChart.vue";
import formatter from "@/mixins/formatter";
import { CURRENCY } from "@/types/evcc";
import { PERIODS } from "../Sessions/types";
import { ENTITY_CHART, type FlowResult, type FlowSink, type FlowSource } from "./types";
import DeviceColorDot from "../Helper/DeviceColorDot.vue";
import { sourceBreakdown, sumEnergy, type SourceRow } from "./mix";

// one entity of a sink group (consumer of home, loadpoint): bars or pattern chart with
// energy, source mix and cost beside it. The header closes when `closable`
export default defineComponent({
	name: "EntityDetail",
	components: {
		Stat,
		GroupChart,
		PatternChart,
		IconSelectGroup,
		IconSelectItem,
		DeviceColorDot,
		CardHeader,
	},
	mixins: [formatter],
	props: {
		title: { type: String, required: true },
		color: { type: String, default: "" },
		sink: { type: String as PropType<FlowSink>, default: "home" },
		price: Number, // per kWh at the sink's rate over the period, undefined without tariffs
		closable: Boolean,
		// bars only, no pattern chart
		chartToggle: { type: Boolean, default: true },
		// the title as a smaller subhead inside a titled card
		subhead: Boolean,
		chartHeight: { type: Number, default: 200 },
		showXAxis: { type: Boolean, default: true },
		// the dot opens the color picker for the device
		pickable: Boolean,
		deviceColor: { type: String, default: "" }, // configured color, empty when automatic
		flow: { type: Object as PropType<FlowResult> },
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
	emits: ["close", "update:chart", "drill", "sources"],
	data() {
		return {
			ENTITY_CHART,
			// below lg the stats wrap under the chart, so stacked charts stop sharing an axis
			mediaQuery: null as MediaQueryList | null,
			isLarge: true,
		};
	},
	mounted() {
		this.mediaQuery = window.matchMedia("(min-width: 992px)");
		this.isLarge = this.mediaQuery.matches;
		this.mediaQuery.addEventListener("change", this.onMediaChange);
	},
	beforeUnmount() {
		this.mediaQuery?.removeEventListener("change", this.onMediaChange);
	},
	methods: {
		onMediaChange(e: MediaQueryListEvent) {
			this.isLarge = e.matches;
		},
		fmtSource(from: FlowSource, share: number): string {
			return `${this.fmtPercentage(share)} ${this.$t(`energy.sources.${from}`)}`;
		},
		fmtMainSource(share: number): string {
			return this.fmtSource(this.mainSource.from, share);
		},
		showSources() {
			this.$emit("sources", { title: this.title, rows: this.sources });
		},
	},
	computed: {
		// with the default aside the energy sits behind the title, the blocks hold sources and cost
		subtitle(): string {
			return this.$slots["aside"] ? "" : this.fmtKWh(this.energy);
		},
		// the sink's source split applied to this entity
		sources(): SourceRow[] {
			return sourceBreakdown(
				this.flow?.flows ?? [],
				this.sink,
				this.energy,
				this.flowSources
			);
		},
		// a site with neither solar nor battery has nothing to tell apart
		hasOwnSources(): boolean {
			return this.sources.some((row) => row.from !== "grid");
		},
		// beside the cost block on small screens, alone at the start without one
		sourcesAlign(): string {
			return this.cost === undefined ? "start" : "end lg-start";
		},
		// the headline: solar, or battery on a site without solar
		mainSource(): SourceRow {
			return this.sources[0]!;
		},
		// the other sources, e.g. "14% Battery, 10% Grid"
		sourceText(): string {
			return this.sources
				.slice(1)
				.map((row) => this.fmtSource(row.from, row.share))
				.join(", ");
		},
		chartOptions(): { value: ENTITY_CHART; name: string; icon: object }[] {
			return [
				{ value: ENTITY_CHART.BARS, icon: BarChartIcon },
				{
					value: ENTITY_CHART.PATTERN,
					icon: FullStackedBarChartIcon,
				},
			].map((o) => ({ ...o, name: this.$t(`energy.consumers.chart.${o.value}`) }));
		},
		energy(): number {
			return this.series ? sumEnergy(this.series) : 0;
		},
		cost(): number | undefined {
			return this.price === undefined ? undefined : this.energy * this.price;
		},
		// grid share at the period's average grid intensity
	},
});
</script>

<style scoped>
@import "../../../css/breakpoints.css";

.header {
	margin-bottom: 1rem;
}
/* the chart's own top space is enough, the plot pulls up into the header row */
@media (--lg-and-up) {
	.header {
		margin-bottom: -0.5rem;
	}
}
.chart-col {
	min-width: 0;
}
/* fixed column beside the chart so the chart width does not depend on the numbers,
   spread between the chart body's top and bottom, above the axis labels */
@media (--lg-and-up) {
	.aside {
		width: 14rem;
		margin-top: -0.5rem;
		padding-bottom: 1.5rem;
	}
}
</style>
