<template>
	<div class="container px-4 safe-area-inset d-flex flex-column">
		<TopHeader :title="`${$t('energy.title')} 🧪`" />
		<div class="row flex-grow-1 d-flex">
			<main class="col-12 d-flex flex-column">
				<PeriodHeader>
					<template #period>
						<PeriodSelector
							:period="effectivePeriod"
							:periodOptions="periodOptions"
							@update:period="changePeriod"
						/>
					</template>
					<template #navigator>
						<DateNavigator
							:day="effectiveDay"
							:month="effectiveMonth"
							:year="effectiveYear"
							:startDate="startDate"
							:showDay="showDayNavigation"
							:showMonth="showMonthNavigation"
							:showYear="showYearNavigation"
							@update-date="updateDate"
						/>
					</template>
				</PeriodHeader>

				<div v-if="!hasData && !loading" class="flex-grow-1 d-flex">
					<div class="empty-box p-5 text-center">
						<span class="text-muted">{{ $t("energy.empty") }}</span>
					</div>
				</div>
				<div v-else class="card-row mb-4">
					<div class="row gx-0 gx-sm-4 gy-4">
						<div class="col-12 col-lg-9 col-xxl-10">
							<Card
								:title="overviewTitle"
								edge-to-edge
								class="h-100"
								data-testid="energy-flow"
								data-anchor
							>
								<template #actions>
									<IconSelectGroup>
										<IconSelectItem
											v-for="option in overviewOptions"
											:key="option.value"
											:active="overview === option.value"
											:title="option.name"
											@click="settings.energyOverview = option.value"
										>
											<component :is="option.icon" />
										</IconSelectItem>
									</IconSelectGroup>
								</template>
								<!-- the sankey fills the height of the bar chart with its legend, so toggling does not jump -->
								<div v-if="overview === OVERVIEW_VIEW.FLOW" class="fill-stack">
									<div class="invisible">
										<div :style="{ height: `${OVERVIEW_HEIGHT}px` }"></div>
										<LegendList class="mt-4" :legends="stackLegends" />
									</div>
									<FlowChart :flows="flows" :loadpoint-labels="loadpointLabels" />
								</div>
								<template v-else>
									<GroupChart
										group="meter"
										:color="colors.grid || ''"
										:series="stackSeries"
										:stack-through="overview === OVERVIEW_VIEW.USAGE"
										:period="effectivePeriod"
										:from="from"
										:to="to"
										single-value
										:height="OVERVIEW_HEIGHT"
										stacked
										auto-range
										:upper-label="$t('energy.flow.pv')"
										:lower-label="$t('energy.flow.withdrawal')"
										@slot="drillDown"
									/>
									<LegendList class="mt-4" :legends="stackLegends" />
								</template>
							</Card>
						</div>
						<div class="col-12 col-lg-3 col-xxl-2">
							<CostStats
								:autarky="autarky"
								:flows="flows"
								:cost="flow?.cost"
								:co2="flow?.co2"
								:currency="currency"
							/>
						</div>
					</div>
				</div>
				<div v-if="hasGrid" class="card-row mb-4">
					<div class="row gx-0 gx-sm-4 gy-4">
						<div class="col-12 col-lg-9 col-xxl-10">
							<Card
								:title="$t('energy.grid.title')"
								edge-to-edge
								class="h-100"
								data-testid="energy-grid"
								data-anchor
							>
								<template #icon>
									<shopicon-regular-powersupply></shopicon-regular-powersupply>
								</template>
								<GroupChart
									group="grid"
									:color="colors.grid || ''"
									:series="gridSeries"
									:prices="chartPrices"
									:currency="currency"
									:height="200"
									:period="effectivePeriod"
									:from="from"
									:to="to"
									@slot="drillDown"
								/>
								<LegendList class="mt-4" :legends="gridLegends" />
							</Card>
						</div>
						<div class="col-12 col-lg-3 col-xxl-2">
							<GridStats :cost="flow?.cost" :currency="currency" />
						</div>
					</div>
				</div>
				<div v-if="pvSeries.length" class="card-row mb-4">
					<div class="row gx-0 gx-sm-4 gy-4">
						<div class="col-12 col-lg-9 col-xxl-10">
							<Card
								:title="$t('energy.production.title')"
								:subtitle="fmtKWh(pvTotal)"
								edge-to-edge
								class="h-100"
								data-testid="energy-production"
								data-anchor
							>
								<template #icon>
									<shopicon-regular-sun></shopicon-regular-sun>
								</template>
								<GroupChart
									group="pv"
									:color="groupColor('pv')"
									:series="pvSeries"
									:overlay="forecastSeries"
									:overlay-color="groupColor('forecast')"
									:overlay-label="$t('energy.group.forecast')"
									:show-overlay="hasForecast && focusedPv === null"
									:height="260"
									:focused-entity="focusedPv"
									:period="effectivePeriod"
									:from="from"
									:to="to"
									@slot="drillDown"
								/>
								<LegendList
									v-if="productionLegends.length"
									class="mt-4"
									:legends="productionLegends"
									@focus="toggleFocusPv"
								/>
							</Card>
						</div>
						<div class="col-12 col-lg-3 col-xxl-2">
							<StatCards :stats="productionStats">
								<template #selfConsumed>
									<MixBar :segments="selfSegments" :tooltip="selfRows" />
									<div class="small text-muted">{{ exportedLabel }}</div>
								</template>
								<template v-if="tomorrowForecast !== undefined" #remaining>
									<Stat
										class="mt-3"
										:number="tomorrowForecast"
										:format="(v: number) => fmtKWh(v)"
										:sub="$t('energy.production.expectedTomorrow')"
									/>
								</template>
								<template v-if="forecastDeviation" #forecast>
									<ForecastDeviation
										class="mt-3"
										:spans="forecastDeviation"
										:color="groupColor('forecast')"
										:labels="bucketLabels"
										:power="isDay"
									/>
								</template>
							</StatCards>
						</div>
					</div>
				</div>
				<Card
					v-for="(bat, i) in batteries"
					:key="bat.title"
					edge-to-edge
					class="box-pull-out mb-4"
					data-testid="energy-battery"
					data-anchor
				>
					<EntityDetail
						:title="bat.title"
						:color="batteryColor(i)"
						:series="{ ...bat, paletteIndex: i }"
						:period="effectivePeriod"
						:from="from"
						:to="to"
						:chart-toggle="false"
						@drill="drillDown"
					>
						<template #icon>
							<shopicon-regular-batterythreequarters></shopicon-regular-batterythreequarters>
						</template>
						<template #aside>
							<div class="col-6 col-lg-12">
								<Stat
									:label="$t('energy.battery.charged')"
									:number="sumEnergy(bat)"
									:format="(v: number) => fmtKWh(v)"
									compact
								/>
							</div>
							<div class="col-6 col-lg-12">
								<Stat
									:label="$t('energy.battery.discharged')"
									align="end lg-start"
									:number="sumEnergy(bat, 'returnEnergy')"
									:format="(v: number) => fmtKWh(v)"
									compact
								/>
							</div>
						</template>
					</EntityDetail>
				</Card>
				<Card
					v-for="lp in loadpoints"
					:key="lp.title"
					edge-to-edge
					class="box-pull-out mb-4"
					data-testid="energy-loadpoint"
					data-anchor
				>
					<EntityDetail
						pickable
						sink="loadpoint"
						:title="lp.title"
						:color="loadpointColors[lp.title] || ''"
						:device-color="deviceColors[lp.title] || ''"
						:price="loadpointPrice"
						:series="lp"
						:daily="loadpointDaily(lp.title) ?? undefined"
						:flow="flow ?? undefined"
						:period="effectivePeriod"
						:from="from"
						:to="to"
						:currency="currency"
						:chart="entityChart"
						@update:chart="settings.energyEntityChart = $event"
						@drill="drillDown"
						:flow-sources="flowSources"
						@sources="showSources"
					/>
				</Card>
				<Card
					v-if="consumers.length"
					:title="$t('energy.consumers.title')"
					:subtitle="consumersSubtitle"
					edge-to-edge
					class="box-pull-out mb-4"
					data-testid="energy-consumers"
					data-anchor
				>
					<template #actions>
						<IconSelectGroup>
							<IconSelectItem
								v-for="option in consumerViewOptions"
								:key="option.value"
								:active="consumerView === option.value"
								:title="option.name"
								@click="setConsumerView(option.value)"
							>
								<component :is="option.icon" />
							</IconSelectItem>
						</IconSelectGroup>
					</template>
					<ConsumerTreemap
						:consumers="consumers"
						:others="others"
						:device-colors="deviceColors"
						:price="homePrice"
						:currency="currency"
						:selected="consumer"
						@select="selectConsumer"
					>
						<template v-if="consumerView === CONSUMER_VIEW.RIVER" #chart>
							<ConsumerRiver
								:layers="riverLayers"
								:selected="consumer"
								:period="effectivePeriod"
								:from="from"
								:to="to"
								@select="selectConsumer"
							/>
						</template>
					</ConsumerTreemap>
					<template v-if="consumer">
						<hr class="my-4" />
						<EntityDetail
							data-anchor
							data-testid="consumer-detail"
							closable
							:pickable="consumerPickable"
							:device-color="deviceColors[consumerTitle] || ''"
							sink="home"
							:price="homePrice"
							:title="consumerTitle"
							:color="consumerColor"
							:flow="flow ?? undefined"
							:series="consumerSeries ?? undefined"
							:period="effectivePeriod"
							:from="from"
							:to="to"
							:currency="currency"
							:daily="consumerDaily ?? undefined"
							:chart="entityChart"
							@update:chart="settings.energyEntityChart = $event"
							@drill="drillDown"
							:flow-sources="flowSources"
							@sources="showSources"
							@close="selectConsumer(null)"
						/>
					</template>
				</Card>
				<!-- no individual consumers: the house consumption alone, nothing to break down -->
				<Card
					v-else-if="homeEnergy > 0"
					edge-to-edge
					class="box-pull-out mb-4"
					data-testid="energy-consumers"
					data-anchor
				>
					<EntityDetail
						sink="home"
						:price="homePrice"
						:title="consumerTitle"
						:color="consumerColor"
						:flow="flow ?? undefined"
						:series="consumerSeries ?? undefined"
						:period="effectivePeriod"
						:from="from"
						:to="to"
						:currency="currency"
						:daily="consumerDaily ?? undefined"
						:chart="entityChart"
						:flow-sources="flowSources"
						@update:chart="settings.energyEntityChart = $event"
						@drill="drillDown"
						@sources="showSources"
					/>
				</Card>
				<MetersCard
					v-if="meters.length"
					:meters="meters"
					:colors="meterColors"
					:device-colors="deviceColors"
					:period="effectivePeriod"
					:from="from"
					:to="to"
					@drill="drillDown"
				/>
				<SourcesModal
					ref="sourcesModal"
					:title="sourcesDetail.title"
					:period="periodLabel"
					:rows="sourcesDetail.rows"
					:currency="currency"
				/>
				<div v-if="hasData" class="d-flex align-items-baseline gap-2 mb-3">
					<DownloadButton :label="$t('general.download')" :href="downloadHref" />
				</div>
			</main>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Header from "../components/Top/Header.vue";
import Card from "../components/Helper/Card.vue";
import "@h2d2/shopicons/es/regular/powersupply";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/batterythreequarters";
import DownloadButton from "../components/Helper/DownloadButton.vue";
import PeriodSelector from "../components/Sessions/PeriodSelector.vue";
import DateNavigator from "../components/Sessions/DateNavigator.vue";
import PeriodHeader from "../components/Sessions/PeriodHeader.vue";
import FlowChart from "../components/Energy/FlowChart.vue";
import GroupChart, { stepAlpha } from "../components/Energy/GroupChart.vue";
import LegendList from "../components/Sessions/LegendList.vue";
import type { Legend } from "../components/Sessions/types";
import IconSelectGroup from "../components/Helper/IconSelectGroup.vue";
import IconSelectItem from "../components/Helper/IconSelectItem.vue";
import BarChartIcon from "../components/MaterialIcon/BarChart.vue";
import SankeyIcon from "../components/MaterialIcon/Sankey.vue";
import PlugIcon from "../components/MaterialIcon/Plug.vue";
import { groupColor } from "../components/Energy/groups";
import { adjustedSolar } from "@/utils/forecast";
import type { TooltipRow } from "../components/Forecast/echarts";
import StatCards from "../components/Energy/StatCards.vue";
import Stat from "../components/Energy/Stat.vue";
import MixBar from "../components/Energy/MixBar.vue";
import GridStats from "../components/Energy/GridStats.vue";
import ForecastDeviation from "../components/Energy/ForecastDeviation.vue";
import CostStats from "../components/Energy/CostStats.vue";
import ConsumerTreemap, { OTHERS } from "../components/Energy/ConsumerTreemap.vue";
import EntityDetail from "../components/Energy/EntityDetail.vue";
import { sinkCost, sumEnergy, type SourceRow } from "../components/Energy/mix";
import { usageSplit } from "../components/Energy/usage";
import { SLOT_MS } from "../components/Energy/slots";
import MetersCard from "../components/Energy/MetersCard.vue";
import SourcesModal from "../components/Energy/SourcesModal.vue";
import ConsumerRiver, { type RiverLayer } from "../components/Energy/ConsumerRiver.vue";
import TreemapIcon from "../components/MaterialIcon/Treemap.vue";
import StreamIcon from "../components/MaterialIcon/Stream.vue";
import type { HistorySeries } from "../components/Energy/GroupChart.vue";
import colors, { batteryColor, darken, deviceColorMap, resolveColors } from "../colors";
import { PERIODS } from "../components/Sessions/types";
import {
	CONSUMER_VIEW,
	ENTITY_CHART,
	OVERVIEW_VIEW,
	type ConsumerEnergy,
	type Flow,
	type FlowResult,
	type FlowSink,
	type FlowSource,
	type StatItem,
	type PriceBand,
	type PriceOverlay,
	type TariffSlot,
} from "../components/Energy/types";
import type { DeviceColors, UiSolarDetails } from "@/types/evcc";
import { CURRENCY } from "@/types/evcc";
import api from "../api";
import settings from "../settings";
import formatter from "../mixins/formatter";
import store from "../store";

const ENERGY_PERIODS = [PERIODS.DAY, PERIODS.MONTH, PERIODS.YEAR];
const OVERVIEW_HEIGHT = 300;

function daysInMonth(year: number, month: number) {
	return new Date(year, month, 0).getDate();
}

function shiftPeriod(period: PERIODS, from: Date, offset: number): Date {
	switch (period) {
		case PERIODS.DAY:
			return new Date(from.getFullYear(), from.getMonth(), from.getDate() + offset);
		case PERIODS.YEAR:
			return new Date(from.getFullYear() + offset, 0, 1);
		case PERIODS.MONTH:
		default:
			return new Date(from.getFullYear(), from.getMonth() + offset, 1);
	}
}

export default defineComponent({
	name: "Energy",
	components: {
		TopHeader: Header,
		Card,
		DownloadButton,
		PeriodSelector,
		DateNavigator,
		PeriodHeader,
		FlowChart,
		GroupChart,
		LegendList,
		IconSelectGroup,
		IconSelectItem,
		StatCards,
		Stat,
		MixBar,
		GridStats,
		ForecastDeviation,
		CostStats,
		ConsumerTreemap,
		EntityDetail,
		MetersCard,
		SourcesModal,
		ConsumerRiver,
	},
	mixins: [formatter],
	props: {
		day: { type: Number, default: undefined },
		month: { type: Number, default: undefined },
		year: { type: Number, default: undefined },
		period: { type: String as PropType<PERIODS>, default: undefined },
		consumer: { type: String, default: null },
		offline: Boolean,
	},
	data() {
		return {
			flow: null as FlowResult | null,
			tariffs: [] as TariffSlot[],
			series: [] as HistorySeries[],
			// daily buckets for the calendar, only fetched for year and total
			dailySeries: [] as HistorySeries[],
			settings,
			colors,
			OVERVIEW_VIEW,
			CONSUMER_VIEW,
			OVERVIEW_HEIGHT,
			loading: false,
			focusedPv: null as number | null,
			sourcesDetail: { title: "", rows: [] as SourceRow[] },
			startDate: new Date(2020, 0, 1),
		};
	},
	head() {
		return { title: this.$t("energy.title") };
	},
	computed: {
		currency(): CURRENCY {
			return store.state.currency || CURRENCY.EUR;
		},
		historyUpdated(): string | undefined {
			return store.state.historyUpdated;
		},
		flows(): Flow[] {
			return this.flow?.flows || [];
		},
		// any metered energy in the period, the flows alone may be empty for partial setups
		hasData(): boolean {
			return this.series.some((s) =>
				s.data.some((slot) => slot.energy > 0 || slot.returnEnergy > 0)
			);
		},
		deviceColors(): DeviceColors {
			return deviceColorMap(store.state.deviceColors);
		},
		entityTotals(): Record<string, ConsumerEnergy[]> {
			const out: Record<string, ConsumerEnergy[]> = {};
			for (const s of this.series) {
				const energy = s.data.reduce((acc, slot) => acc + slot.energy, 0);
				(out[s.group] ||= []).push({ title: s.title, energy });
			}
			return out;
		},
		consumers(): ConsumerEnergy[] {
			return this.entityTotals["consumer"] || [];
		},
		homeEnergy(): number {
			return (this.entityTotals["home"] || []).reduce((acc, e) => acc + e.energy, 0);
		},
		// metered energy at the average price of the tariff-covered slots, same basis as the legend
		consumersSubtitle(): string {
			const parts = [this.fmtKWh(this.homeEnergy)];
			if (this.homePrice !== undefined) {
				parts.push(
					this.fmtMoneyWithSymbol(this.homeEnergy * this.homePrice, this.currency)
				);
				parts.push(`ø ${this.fmtPricePerKWh(this.homePrice, this.currency)}`);
			}
			return parts.join(" · ");
		},
		// the selected day, month or year
		periodLabel(): string {
			switch (this.effectivePeriod) {
				case PERIODS.DAY:
					return this.fmtDayMonthYear(this.from);
				case PERIODS.MONTH:
					return this.fmtMonthYear(this.from);
				default:
					return String(this.from.getFullYear());
			}
		},
		// sources with an entity, in attribution order
		flowSources(): FlowSource[] {
			const sources: FlowSource[] = [];
			if (this.pvSeries.length) sources.push("pv");
			if (this.batteries.length) sources.push("battery");
			if (this.hasGrid) sources.push("grid");
			return sources;
		},
		// average price of home consumption, applied to every consumer
		homePrice(): number | undefined {
			return this.sinkPrice("home");
		},
		// loadpoints are what consumption prices beyond home
		loadpointPrice(): number | undefined {
			return this.sinkPrice("loadpoint");
		},
		// unknown stored values fall back to the treemap
		consumerView(): CONSUMER_VIEW {
			const stored = settings.energyConsumerView as CONSUMER_VIEW;
			return Object.values(CONSUMER_VIEW).includes(stored) ? stored : CONSUMER_VIEW.TREEMAP;
		},
		consumerViewOptions(): { value: CONSUMER_VIEW; name: string; icon: object }[] {
			return [
				{ value: CONSUMER_VIEW.TREEMAP, icon: TreemapIcon },
				{ value: CONSUMER_VIEW.RIVER, icon: StreamIcon },
			].map((o) => ({ ...o, name: this.$t(`energy.consumers.view.${o.value}`) }));
		},
		// the consumers over time in the treemap's colors: others first, then by energy
		riverLayers(): RiverLayer[] {
			const palette = this.consumerPalette;
			const layers: RiverLayer[] = this.withData("consumer")
				.map((s) => ({
					key: s.title,
					name: s.title,
					color: palette[s.title] || "",
					series: s,
				}))
				.sort((a, b) => sumEnergy(b.series) - sumEnergy(a.series));
			const others = this.othersOf(this.series);
			if (others && this.others > 0) {
				layers.unshift({
					key: OTHERS,
					name: this.$t("energy.consumers.others"),
					color: colors.muted || "",
					series: others,
				});
			}
			return layers;
		},
		// the consumer shown in detail: the selected one, or all of the house
		// consumption when there are no individual consumers to pick from
		activeConsumer(): string | null {
			return this.consumers.length ? this.consumer : OTHERS;
		},
		consumerTitle(): string {
			if (!this.consumers.length) return this.$t("energy.group.consumer");
			return this.activeConsumer === OTHERS
				? this.$t("energy.consumers.others")
				: (this.activeConsumer ?? "");
		},
		consumerPickable(): boolean {
			return this.activeConsumer !== OTHERS;
		},
		consumerColor(): string {
			if (this.activeConsumer === OTHERS) return colors.muted || "";
			return this.consumerPalette[this.activeConsumer ?? ""] || "";
		},
		// sorted so a consumer keeps its color, the treemap resolves the same way
		consumerPalette(): Record<string, string> {
			return resolveColors(this.consumers.map((e) => e.title).sort(), this.deviceColors);
		},
		consumerSeries(): HistorySeries | null {
			return this.entitySeries(this.series);
		},
		// unknown stored values fall back to the default
		overview(): OVERVIEW_VIEW {
			const stored = settings.energyOverview as OVERVIEW_VIEW;
			return Object.values(OVERVIEW_VIEW).includes(stored) ? stored : OVERVIEW_VIEW.OVERVIEW;
		},
		isDay(): boolean {
			return this.effectivePeriod === PERIODS.DAY;
		},
		overviewOptions(): { value: OVERVIEW_VIEW; name: string; icon: object }[] {
			return [
				{
					value: OVERVIEW_VIEW.OVERVIEW,
					icon: BarChartIcon,
					name: this.$t("energy.flow.overview"),
				},
				{ value: OVERVIEW_VIEW.USAGE, icon: PlugIcon, name: this.$t("energy.flow.usage") },
				{ value: OVERVIEW_VIEW.FLOW, icon: SankeyIcon, name: this.$t("energy.flow.title") },
			];
		},
		overviewTitle(): string {
			return this.overviewOptions.find((o) => o.value === this.overview)?.name || "";
		},
		// the buckets every stack is built on
		flowSlots() {
			return this.series.find((s) => s.group === "grid" || s.group === "pv")?.data || [];
		},
		stackSeries(): HistorySeries[] {
			return this.overview === OVERVIEW_VIEW.USAGE ? this.usageSeries : this.flowSeries;
		},
		// usage: the overview's column, from minus import and discharge up to production,
		// as one stack of sinks running through the baseline: batteries at the bottom,
		// then loadpoints, consumption, export on top. Amounts per sink from usageSplit
		usageSeries(): HistorySeries[] {
			const pv = this.slotSum("pv", "energy");
			const gridIn = this.slotSum("grid", "energy");
			const gridOut = this.slotSum("grid", "returnEnergy");
			const batOut = this.slotSum("battery", "returnEnergy");
			const home = this.slotSum("home", "energy");
			const perEntity = (s: HistorySeries) => {
				const m = new Map(s.data.map((slot) => [slot.start, slot.energy]));
				return (t: string) => m.get(t) || 0;
			};
			const lps = this.loadpoints.map(perEntity);
			const bats = this.batteries.map(perEntity);
			const slots = this.flowSlots;
			const input = slots.map(({ start: t }) => ({
				pv: pv(t),
				gridIn: gridIn(t),
				gridOut: gridOut(t),
				batOut: batOut(t),
				home: home(t),
				bats: bats.map((b) => b(t)),
				lps: lps.map((l) => l(t)),
			}));
			// per slot each sink's share above and below the baseline in stack order
			const bands = usageSplit(input).map((r, t) => {
				let level = -(input[t]!.gridIn + input[t]!.batOut);
				return [...r.bats, ...r.lps, r.home, r.export].map((a) => {
					const start = level;
					level += a;
					return [
						Math.max(0, level) - Math.max(0, start),
						Math.min(0, level) - Math.min(0, start),
					];
				});
			});
			const entries: [string, string][] = [
				...this.batteries.map((s, i): [string, string] => [s.title, batteryColor(i)]),
				...this.loadpoints.map((s): [string, string] => [
					s.title,
					this.loadpointColors[s.title] || "",
				]),
				[this.$t("energy.flow.home"), colors.muted || ""],
				[this.$t("energy.flow.export"), colors.export || ""],
			];
			return entries.map(([title, color], i) => ({
				title,
				group: "meter",
				color,
				data: slots.map((slot, t) => {
					const [energy, returnEnergy] = bands[t]![i]!;
					return { start: slot.start, end: slot.end, energy, returnEnergy };
				}),
			}));
		},
		// per bucket: production above the axis split into self-use, battery charging
		// and grid export; below the axis what production could not cover, battery
		// discharge and grid import. Consumption is self-use plus the bands below.
		flowSeries(): HistorySeries[] {
			const slots = this.flowSlots;
			const pv = this.slotSum("pv", "energy");
			const gridIn = this.slotSum("grid", "energy");
			const gridOut = this.slotSum("grid", "returnEnergy");
			const batIn = this.slotSum("battery", "energy");
			const batOut = this.slotSum("battery", "returnEnergy");
			const build = (
				title: string,
				color: string,
				up: (start: string) => number,
				down: (start: string) => number = () => 0
			) => ({
				title,
				group: "meter",
				color,
				data: slots.map((slot) => ({
					start: slot.start,
					end: slot.end,
					energy: up(slot.start),
					returnEnergy: down(slot.start),
				})),
			});
			// production covers export first, then battery charging, the rest is self-use
			const exportUp = (t: string) => Math.min(pv(t), gridOut(t));
			const batUp = (t: string) => Math.min(batIn(t), Math.max(0, pv(t) - gridOut(t)));
			const selfUp = (t: string) => Math.max(0, pv(t) - gridOut(t) - batIn(t));
			return [
				build(this.$t("energy.flow.batteryCharge"), groupColor("battery"), batUp),
				build(
					this.$t("energy.flow.batteryDischarge"),
					groupColor("battery"),
					() => 0,
					batOut
				),
				build(this.$t("main.energyflow.selfConsumption"), groupColor("pv"), selfUp),
				build(this.$t("energy.flow.grid"), colors.grid || "", () => 0, gridIn),
				build(this.$t("energy.flow.export"), colors.export || "", exportUp),
			];
		},
		stackLegends(): Legend[] {
			const total = (s: HistorySeries) =>
				s.data.reduce((acc, slot) => acc + slot.energy + slot.returnEnergy, 0);
			const [charge, discharge, self, gridImport, gridExport] = this.flowSeries;
			// both read top down like their stack
			const ordered =
				this.overview === OVERVIEW_VIEW.USAGE
					? [...this.usageSeries].reverse()
					: [gridExport, self, charge, discharge, gridImport];
			return ordered
				.filter((s): s is HistorySeries => !!s)
				.map((s) => ({
					label: s.title,
					color: s.color || "",
					value: this.fmtKWh(total(s)),
				}));
		},
		batteries(): HistorySeries[] {
			return this.withData("battery");
		},
		meters(): HistorySeries[] {
			return this.withData("meter");
		},
		meterColors(): Record<string, string> {
			return this.entityColors(this.meters);
		},
		loadpoints(): HistorySeries[] {
			return this.withData("loadpoint");
		},
		loadpointLabels(): string[] {
			const charging = this.loadpoints.some((lp) => !lp.isTemp);
			const heating = this.loadpoints.some((lp) => lp.isTemp);
			const both = charging === heating;
			return [
				...(charging || both ? [this.$t("energy.flow.charging")] : []),
				...(heating || both ? [this.$t("energy.flow.heating")] : []),
			];
		},
		loadpointColors(): Record<string, string> {
			return this.entityColors(this.loadpoints);
		},
		pvSeries(): HistorySeries[] {
			return this.withData("pv");
		},
		forecastSeries(): HistorySeries[] {
			return this.series.filter((s) => s.group === "forecast");
		},
		hasForecast(): boolean {
			return this.forecastSeries.some((s) => s.data.some((slot) => slot.energy > 0));
		},
		pvTotal(): number {
			return this.total(this.pvSeries);
		},
		forecastTotal(): number {
			return this.total(this.forecastSeries);
		},
		autarky(): number {
			const consumption = this.sumFlows((f) => f.to === "home" || f.to === "loadpoint");
			return this.ratio(1 - this.sumFlows((f) => f.from === "grid") / consumption);
		},
		gridSeries(): HistorySeries[] {
			return this.withData("grid");
		},
		// with the price overlay on, both prices as lines with their range over the period
		gridLegends(): Legend[] {
			const list: Legend[] = [
				{
					label: this.$t("energy.grid.import"),
					color: colors.grid || "",
					value: this.fmtKWh(this.gridImport),
				},
				{
					label: this.$t("energy.grid.export"),
					color: colors.export || "",
					value: this.fmtKWh(this.gridExport),
				},
			];
			const prices = this.gridPrices;
			const range = (band: PriceBand) => {
				const known = (v: (number | null)[]) => v.filter((x): x is number => x !== null);
				return this.fmtPriceRange(
					Math.min(...known(band.lo)),
					Math.max(...known(band.hi)),
					this.currency
				);
			};
			// a day shows the tariff's range, longer periods what was paid on average
			const cost = this.flow?.cost;
			const price = (band: PriceBand, amount?: number, energy?: number) =>
				this.isDay || !amount || !energy
					? range(band)
					: `ø ${this.fmtPricePerKWh(amount / energy, this.currency)}`;
			if (prices?.import) {
				list.push({
					label: this.$t("energy.grid.importPrice"),
					color: colors.price || "",
					value: price(prices.import, cost?.import, cost?.importEnergy),
					type: "line",
				});
			}
			if (prices?.feedin) {
				list.push({
					label: this.$t("energy.grid.exportPrice"),
					color: colors.export || "",
					value: price(prices.feedin, cost?.export, cost?.exportEnergy),
					type: "line",
				});
			}
			return list;
		},
		// per bucket for chart and tooltip: what the energy cost or earned, and its price.
		// A day's slots carry the tariff, longer buckets the effective price of their
		// energy, the tariff average where none flowed
		chartPrices(): PriceOverlay | null {
			const tariffs = this.gridPrices;
			if (!tariffs) return null;
			type Key = "cost" | "pricedEnergy" | "returnCost" | "pricedReturnEnergy";
			const sums = (key: Key) => {
				const out = Array.from({ length: this.bucketCount }, (): number | null => null);
				for (const s of this.gridSeries) {
					for (const slot of s.data) {
						const v = slot[key];
						if (v == null) continue;
						const i = this.bucketIndex(slot.start);
						out[i] = (out[i] ?? 0) + v;
					}
				}
				return out;
			};
			const band = (tariff: PriceBand | undefined, costKey: Key, energyKey: Key) => {
				if (!tariff) return undefined;
				const cost = sums(costKey);
				if (this.effectivePeriod === PERIODS.DAY) return { ...tariff, cost };
				const energy = sums(energyKey);
				// by priced energy, not all energy: unpriced slots must not dilute it
				const avg = tariff.avg.map((price, i) =>
					energy[i] ? cost[i]! / energy[i]! : price
				);
				return { avg, lo: avg, hi: avg, cost };
			};
			return {
				import: band(tariffs.import, "cost", "pricedEnergy"),
				feedin: band(tariffs.feedin, "returnCost", "pricedReturnEnergy"),
			};
		},
		// the tariff per bucket, its average and range
		gridPrices(): PriceOverlay | null {
			if (!this.tariffs.length) return null;
			const band = (key: "grid" | "feedin"): PriceBand | undefined => {
				const empty = () =>
					Array.from({ length: this.bucketCount }, (): number | null => null);
				const out = { avg: empty(), lo: empty(), hi: empty() };
				for (const t of this.tariffs) {
					const v = t[key];
					if (v === undefined) continue;
					const i = this.bucketIndex(t.start);
					out.avg[i] = v;
					out.lo[i] = t[`${key}Min`] ?? v;
					out.hi[i] = t[`${key}Max`] ?? v;
				}
				return out.avg.some((v) => v !== null) ? out : undefined;
			};
			return { import: band("grid"), feedin: band("feedin") };
		},
		gridImport(): number {
			return this.gridSeries.reduce((acc, s) => acc + sumEnergy(s), 0);
		},
		gridExport(): number {
			return this.gridSeries.reduce((acc, s) => acc + sumEnergy(s, "returnEnergy"), 0);
		},
		// no grid meter rows in the period, the cards would only show zeros
		hasGrid(): boolean {
			return this.series.some(
				(s) =>
					s.group === "grid" &&
					s.data.some((slot) => slot.energy > 0 || slot.returnEnergy > 0)
			);
		},
		// tooltip headline per bucket: slot, day or month
		bucketLabels(): string[] {
			const count = this.buckets("grid", "energy").length;
			return Array.from({ length: count }, (_, i) => {
				switch (this.effectivePeriod) {
					case PERIODS.DAY:
						return this.fmtTimeSlot(
							new Date(this.from.getTime() + i * SLOT_MS),
							SLOT_MS
						);
					case PERIODS.MONTH:
						return this.fmtDayMonth(
							new Date(this.effectiveYear, this.effectiveMonth - 1, i + 1)
						);
					default:
						return this.fmtMonthYear(new Date(this.effectiveYear, i, 1));
				}
			});
		},
		// per bucket the forecast and what was produced
		forecastDeviation(): ([number, number] | null)[] | null {
			if (!this.hasForecast) return null;
			const pv = this.buckets("pv", "energy");
			// a bucket without a persisted forecast slot (restart gap) has no delta to show
			const known = new Set(
				this.forecastSeries.flatMap((s) =>
					s.data.map((slot) => this.bucketIndex(slot.start))
				)
			);
			return this.buckets("forecast", "energy").map((f, i): [number, number] | null =>
				known.has(i) ? [f, pv[i] || 0] : null
			);
		},
		bucketCount(): number {
			switch (this.effectivePeriod) {
				case PERIODS.DAY:
					return 96;
				case PERIODS.MONTH:
					return daysInMonth(this.effectiveYear, this.effectiveMonth);
				default:
					return 12;
			}
		},
		selfConsumption(): number {
			return (
				1 - this.sumFlows((f) => f.to === "export") / this.sumFlows((f) => f.from === "pv")
			);
		},
		// today's day view is still running, show what the forecast still expects
		inFlightDay(): boolean {
			return (
				this.effectivePeriod === PERIODS.DAY &&
				this.from.toDateString() === new Date().toDateString()
			);
		},
		solar(): UiSolarDetails | undefined {
			const solar = store.uiForecast.value.solar;
			return store.state.solarAdjusted ? adjustedSolar(solar) : solar;
		},
		// forecast energies are Wh
		remainingForecast(): number | undefined {
			const today = this.solar?.today;
			return today ? today.energy / 1e3 : undefined;
		},
		tomorrowForecast(): number | undefined {
			const tomorrow = this.solar?.tomorrow;
			return tomorrow ? tomorrow.energy / 1e3 : undefined;
		},
		// production split into what was used and what went to the grid
		selfShare(): number {
			return this.ratio(this.selfConsumption);
		},
		selfSegments(): { value: number; color: string }[] {
			return [
				{ value: this.selfShare, color: groupColor("pv") },
				{ value: 100 - this.selfShare, color: colors.export || "" },
			];
		},
		// the rest of the production
		exportedLabel(): string {
			return this.$t("energy.production.exported", {
				value: this.fmtPercentage(100 - this.selfShare),
			});
		},
		selfRows(): TooltipRow[] {
			const produced = this.sumFlows((f) => f.from === "pv");
			const exported = this.sumFlows((f) => f.to === "export");
			const row = (name: string, share: number, energy: number) => ({
				name,
				values: [this.fmtPercentage(share), this.fmtKWh(energy)],
			});
			return [
				row(this.$t("energy.production.selfConsumed"), this.selfShare, produced - exported),
				row(this.$t("energy.flow.export"), 100 - this.selfShare, exported),
			];
		},
		productionStats(): StatItem[] {
			const list: StatItem[] = [
				{
					key: "selfConsumed",
					label: this.$t("energy.production.selfConsumed"),
					number: this.selfShare,
					format: (v: number) => this.fmtPercentage(v),
				},
			];
			if (this.inFlightDay && this.remainingForecast !== undefined) {
				// own key: a separate card without the accuracy chart and a fresh number animation
				list.push({
					key: "remaining",
					label: this.$t("energy.group.forecast"),
					number: this.remainingForecast,
					format: (v: number) => this.fmtKWh(v),
					sub: this.$t("energy.production.remainingToday"),
				});
			} else if (this.forecastTotal > 0) {
				// absolute error per bucket, so misses in both directions do not cancel out
				const error = (this.forecastDeviation || []).reduce(
					(acc, s) => acc + (s ? Math.abs(s[0] - s[1]) : 0),
					0
				);
				const accuracy = 1 - error / this.forecastTotal;
				const delta = this.forecastTotal - this.pvTotal;
				list.push({
					key: "forecast",
					label: this.$t("energy.production.accuracy"),
					number: this.ratio(accuracy),
					format: (v: number) => this.fmtPercentage(v),
					// how far the forecast total sits above or below what was produced
					sub: this.$t(
						delta >= 0 ? "energy.production.above" : "energy.production.below",
						{
							value: this.fmtKWh(Math.abs(delta)),
						}
					),
				});
			}
			return list;
		},
		// entities only when there is more than one, forecast as a line entry
		productionLegends(): Legend[] {
			const n = this.pvSeries.length;
			const list: Legend[] =
				n > 1
					? this.pvSeries.map((s, i) => ({
							label: s.title,
							color: darken(groupColor("pv"), stepAlpha(i, n)),
							value: this.fmtKWh(this.total([s])),
							focusable: true,
							focusKey: i,
							dim: this.focusedPv !== null && this.focusedPv !== i,
						}))
					: [];
			if (this.hasForecast) {
				list.push({
					label: this.$t("energy.group.forecast"),
					color: groupColor("forecast"),
					value: this.fmtKWh(this.forecastTotal),
					type: "line",
					dim: this.focusedPv !== null,
				});
			}
			return list;
		},
		entityChart(): ENTITY_CHART {
			const stored = settings.energyEntityChart as ENTITY_CHART;
			return Object.values(ENTITY_CHART).includes(stored) ? stored : ENTITY_CHART.BARS;
		},
		// month buckets are days already, year and total need the extra daily fetch
		// the day view already has 15 minute buckets
		consumerDaily(): HistorySeries | null {
			if (this.effectivePeriod === PERIODS.DAY) return this.consumerSeries;
			return this.entitySeries(this.dailySeries);
		},
		// the pattern chart needs finer buckets than the period view for the open
		// consumer and every loadpoint
		dailyKey(): string {
			const calendar =
				this.entityChart === ENTITY_CHART.PATTERN && this.effectivePeriod !== PERIODS.DAY;
			const titles = [this.activeConsumer, ...this.loadpoints.map((s) => s.title)].filter(
				Boolean
			);
			return calendar && titles.length ? `${this.fetchKey}|${titles.join(",")}` : "";
		},
		// csv of the period's series, same query as the charts
		downloadHref(): string {
			const params = new URLSearchParams({
				lang: this.$i18n?.locale,
				from: this.from.toISOString(),
				to: this.to.toISOString(),
				aggregate: this.aggregate,
			});
			return `./api/history/energy?${params.toString()}`;
		},
		aggregate(): string {
			switch (this.effectivePeriod) {
				case PERIODS.DAY:
					return "15m";
				case PERIODS.MONTH:
					return "day";
				default:
					return "month";
			}
		},
		others(): number {
			return (
				this.othersOf(this.series)?.data.reduce((acc, slot) => acc + slot.energy, 0) || 0
			);
		},

		effectivePeriod(): PERIODS {
			return this.period && ENERGY_PERIODS.includes(this.period)
				? (this.period as PERIODS)
				: PERIODS.DAY;
		},
		effectiveYear(): number {
			return this.year ?? new Date().getFullYear();
		},
		effectiveMonth(): number {
			return this.month ?? new Date().getMonth() + 1;
		},
		effectiveDay(): number {
			const fallback = new Date().getDate();
			const d = this.day ?? fallback;
			return Math.min(d, daysInMonth(this.effectiveYear, this.effectiveMonth));
		},
		periodOptions() {
			return ENERGY_PERIODS.map((value) => ({
				name: this.$t(`sessions.period.${value}`),
				value,
			}));
		},
		showDayNavigation(): boolean {
			return this.effectivePeriod === PERIODS.DAY;
		},
		showMonthNavigation(): boolean {
			return this.effectivePeriod === PERIODS.MONTH;
		},
		showYearNavigation(): boolean {
			return [PERIODS.MONTH, PERIODS.YEAR].includes(this.effectivePeriod);
		},
		from(): Date {
			switch (this.effectivePeriod) {
				case PERIODS.DAY:
					return new Date(this.effectiveYear, this.effectiveMonth - 1, this.effectiveDay);
				case PERIODS.YEAR:
					return new Date(this.effectiveYear, 0, 1);
				default:
					return new Date(this.effectiveYear, this.effectiveMonth - 1, 1);
			}
		},
		to(): Date {
			return shiftPeriod(this.effectivePeriod, this.from, 1);
		},
		fetchKey(): string {
			return `${this.from.getTime()}|${this.to.getTime()}`;
		},
	},
	watch: {
		fetchKey() {
			this.fetchData();
		},
		dailyKey() {
			this.fetchDaily();
		},
		offline(offline) {
			if (!offline) this.fetchData();
		},
		historyUpdated() {
			this.fetchData();
		},
	},
	mounted() {
		this.fetchData();
		this.fetchDaily();
	},
	methods: {
		batteryColor,
		sumEnergy,
		groupColor,
		// one value per slot, day or month spanning the whole period, so a running day
		// stays empty on the right
		buckets(group: string, key: "energy" | "returnEnergy"): number[] {
			const out = Array.from({ length: this.bucketCount }, () => 0);
			for (const s of this.series.filter((s) => s.group === group)) {
				for (const slot of s.data) {
					const i = this.bucketIndex(slot.start);
					out[i] = (out[i] || 0) + slot[key];
				}
			}
			return out;
		},
		bucketIndex(start: string): number {
			const d = new Date(start);
			switch (this.effectivePeriod) {
				case PERIODS.DAY:
					return d.getHours() * 4 + Math.floor(d.getMinutes() / 15);
				case PERIODS.MONTH:
					return d.getDate() - 1;
				default:
					return d.getMonth();
			}
		},
		total(list: HistorySeries[]): number {
			return list.reduce((acc, s) => acc + sumEnergy(s), 0);
		},
		// entities of a group that metered anything in the period
		withData(group: string): HistorySeries[] {
			return this.series.filter(
				(s) =>
					s.group === group &&
					s.data.some((slot) => slot.energy > 0 || slot.returnEnergy > 0)
			);
		},
		// energy per bucket summed over the entities of a group
		slotSum(group: string, key: "energy" | "returnEnergy"): (start: string) => number {
			const out = new Map<string, number>();
			for (const s of this.series.filter((s) => s.group === group)) {
				for (const slot of s.data)
					out.set(slot.start, (out.get(slot.start) || 0) + slot[key]);
			}
			return (start) => out.get(start) || 0;
		},
		entityColors(list: HistorySeries[]): Record<string, string> {
			const titles = list.map((s) => s.title).sort();
			return resolveColors(titles, this.deviceColors);
		},
		sumFlows(match: (f: Flow) => boolean): number {
			return this.flows.filter(match).reduce((acc, f) => acc + f.energy, 0);
		},
		ratio(v: number): number {
			return Number.isFinite(v) ? Math.min(100, Math.max(0, v * 100)) : 0;
		},
		toggleFocusPv(legend: Legend) {
			if (legend.focusKey === undefined) return;
			const key = legend.focusKey as number;
			this.focusedPv = this.focusedPv === key ? null : key;
		},
		async fetchData() {
			this.loading = true;
			const requestKey = this.fetchKey;
			try {
				const [flow, energy, tariffs] = await Promise.all([
					this.fetchFlow(this.from, this.to),
					this.fetchEnergy(this.from, this.to),
					this.fetchTariffs(this.from, this.to),
				]);
				// a newer request superseded this one
				if (requestKey !== this.fetchKey) return;
				const anchor = this.scrollAnchor();
				this.flow = flow.data || null;
				this.series = energy.data || [];
				this.tariffs = tariffs.data || [];
				if (this.focusedPv !== null && this.focusedPv >= this.pvSeries.length)
					this.focusedPv = null;
				await this.$nextTick();
				this.restoreScroll(anchor);
				this.prefetchAdjacent();
			} catch (e) {
				console.error("Failed to load energy flow", e);
			} finally {
				if (requestKey === this.fetchKey) this.loading = false;
			}
		},
		fetchFlow(from: Date, to: Date) {
			return api.get("history/flow", {
				params: { from: from.toISOString(), to: to.toISOString() },
			});
		},
		fetchTariffs(from: Date, to: Date) {
			return api.get("history/tariffs", {
				params: {
					from: from.toISOString(),
					to: to.toISOString(),
					aggregate: this.aggregate,
				},
			});
		},
		fetchEnergy(from: Date, to: Date) {
			return api.get("history/energy", {
				params: {
					from: from.toISOString(),
					to: to.toISOString(),
					aggregate: this.aggregate,
				},
			});
		},
		// home minus all consumers per bucket, clamped so meter noise never goes negative
		othersOf(series: HistorySeries[]): HistorySeries | null {
			const home = series.find((s) => s.group === "home");
			if (!home) return null;
			const tracked = new Map<string, number>();
			for (const s of series.filter((s) => s.group === "consumer")) {
				for (const slot of s.data)
					tracked.set(slot.start, (tracked.get(slot.start) || 0) + slot.energy);
			}
			return {
				title: this.$t("energy.consumers.others"),
				group: "consumer",
				virtual: true,
				data: home.data.map((slot) => ({
					...slot,
					energy: Math.max(0, slot.energy - (tracked.get(slot.start) || 0)),
				})),
			};
		},
		loadpointDaily(title: string): HistorySeries | null {
			const list = this.effectivePeriod === PERIODS.DAY ? this.series : this.dailySeries;
			return list.find((s) => s.group === "loadpoint" && s.title === title) || null;
		},
		entitySeries(series: HistorySeries[]): HistorySeries | null {
			if (!this.activeConsumer) return null;
			if (this.activeConsumer === OTHERS) return this.othersOf(series);
			return (
				series.find((s) => s.group === "consumer" && s.title === this.activeConsumer) ||
				null
			);
		},
		async fetchDaily() {
			if (!this.dailyKey) {
				this.dailySeries = [];
				return;
			}
			const requestKey = this.dailyKey;
			const base = {
				from: this.from.toISOString(),
				to: this.to.toISOString(),
				// hours for a month, days for year and total
				aggregate: this.effectivePeriod === PERIODS.MONTH ? "hour" : "day",
			};
			// the consumer itself, others needs home and all consumers, plus the loadpoints
			const scopes: Record<string, string>[] = [];
			if (this.activeConsumer === OTHERS)
				scopes.push({ group: "home" }, { group: "consumer" });
			else if (this.activeConsumer) scopes.push({ title: this.activeConsumer });
			if (this.loadpoints.length) scopes.push({ group: "loadpoint" });
			try {
				const res = await Promise.all(
					scopes.map((scope) =>
						api.get("history/energy", { params: { ...base, ...scope } })
					)
				);
				if (requestKey !== this.dailyKey) return;
				this.dailySeries = res.flatMap((r) => r.data || []);
			} catch (e) {
				console.error("Failed to load daily energy", e);
			}
		},
		// Keep the first visible card or section where it is while the content above
		// it changes size. The browser's own anchoring loses track when the legend
		// rows are re-rendered.
		scrollAnchor(): { el: Element; top: number } | null {
			const anchors = [...document.querySelectorAll("[data-anchor]")];
			const visible = anchors.find((el) => el.getBoundingClientRect().top >= 0);
			const el = visible ?? anchors.at(-1);
			return el ? { el, top: el.getBoundingClientRect().top } : null;
		},
		restoreScroll(anchor: { el: Element; top: number } | null) {
			if (!anchor || !anchor.el.isConnected) return;
			const delta = anchor.el.getBoundingClientRect().top - anchor.top;
			if (Math.abs(delta) > 1) window.scrollBy({ top: delta, behavior: "instant" });
		},
		// a second click on the open entity closes it, like the production legend
		// another view of the consumers starts without a selection
		setConsumerView(view: CONSUMER_VIEW) {
			settings.energyConsumerView = view;
			if (this.consumer) this.selectConsumer(null);
		},
		async selectConsumer(key: string | null) {
			const next = key === this.consumer ? null : key;
			const query = this.buildBaseQuery();
			if (next) query["consumer"] = next;
			else delete query["consumer"];
			await this.$router.push({ query });
			if (next) this.revealDetail();
		},
		// bring the detail section into view once when it opens below the fold,
		// leaving room for the sticky period header
		revealDetail() {
			const el = document.querySelector('[data-testid="consumer-detail"]');
			if (!el) return;
			const top = el.getBoundingClientRect().top;
			const header =
				document.querySelector(".sticky-top")?.getBoundingClientRect().bottom ?? 0;
			if (top > window.innerHeight - 160) {
				window.scrollBy({ top: top - header - 24, behavior: "smooth" });
			}
		},
		// a bar in the year view opens that month, a day bar opens that day
		// average price of the priced energy that reached the sink, undefined without any
		sinkPrice(sink: FlowSink): number | undefined {
			const { cost, energy } = sinkCost(this.flows, sink);
			return energy ? cost / energy : undefined;
		},
		showSources(detail: { title: string; rows: SourceRow[] }) {
			this.sourcesDetail = detail;
			(this.$refs["sourcesModal"] as unknown as { open: () => void }).open();
		},
		drillDown(start: Date) {
			if (this.effectivePeriod === PERIODS.DAY) return;
			const query = this.buildBaseQuery();
			query["year"] = String(start.getFullYear());
			query["month"] = String(start.getMonth() + 1);
			if (this.effectivePeriod === PERIODS.MONTH) {
				query["day"] = String(start.getDate());
				delete query["period"];
			} else {
				query["period"] = PERIODS.MONTH;
				delete query["day"];
			}
			this.$router.push({ query });
		},
		// warm the browser http cache for the prev/next period
		prefetchAdjacent() {
			for (const offset of [-1, 1]) {
				const from = shiftPeriod(this.effectivePeriod, this.from, offset);
				const to = shiftPeriod(this.effectivePeriod, this.from, offset + 1);
				if (from > new Date() || to <= this.startDate) continue;
				this.fetchFlow(from, to).catch(() => {});
				this.fetchEnergy(from, to).catch(() => {});
			}
		},
		buildBaseQuery(): Record<string, string | undefined> {
			const out: Record<string, string | undefined> = {};
			for (const [k, v] of Object.entries(this.$route.query)) {
				if (typeof v === "string") out[k] = v;
			}
			return out;
		},
		changePeriod(newPeriod: PERIODS) {
			const query = this.buildBaseQuery();
			query["period"] = newPeriod === PERIODS.DAY ? undefined : newPeriod;
			const now = new Date();
			if (newPeriod === PERIODS.DAY) {
				query["year"] = String(this.effectiveYear);
				query["month"] = String(this.effectiveMonth);
				query["day"] = String(this.day ?? now.getDate());
			} else if (newPeriod === PERIODS.MONTH) {
				query["year"] = String(this.effectiveYear);
				query["month"] = String(this.effectiveMonth);
				delete query["day"];
			} else {
				query["year"] = String(this.effectiveYear);
				delete query["month"];
				delete query["day"];
			}
			this.$router.push({ query });
		},
		updateDate({ year, month, day }: { year: number; month?: number; day?: number }) {
			const query = this.buildBaseQuery();
			query["year"] = String(year);
			if (month !== undefined) query["month"] = String(month);
			else delete query["month"];
			if (day !== undefined) query["day"] = String(day);
			else if (this.effectivePeriod !== PERIODS.DAY) delete query["day"];
			this.$router.push({ query });
		},
	},
});
</script>

<style scoped>
/* rows of cards: pulled into the container padding like a single card from sm up,
   flush on phones where the edge-to-edge cards bleed on their own */
@media (min-width: 576px) {
	.card-row {
		margin-left: -1.5rem;
		margin-right: -1.5rem;
	}
}
/* overlay: the sizer sets the height, the chart fills it */
.fill-stack {
	display: grid;
}
.fill-stack > * {
	grid-area: 1 / 1;
	min-width: 0;
}
</style>
