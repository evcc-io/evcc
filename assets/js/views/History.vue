<template>
	<div
		class="container px-4 safe-area-inset d-flex flex-column"
		:class="{ 'empty-container': !visibleGroups.length }"
	>
		<TopHeader :title="$t('main.history.title')" />
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

				<p class="text-muted small history-disclaimer mb-4">
					Work in progress. Visualizations will change. The current focus is verifying
					that the data is correct. Please report anything implausible.
				</p>

				<div v-if="loading && !visibleGroups.length" class="skeleton-stack">
					<div
						v-for="i in 3"
						:key="i"
						class="history-tile history-tile-skeleton mb-4"
						aria-hidden="true"
					></div>
				</div>
				<div v-else-if="!visibleGroups.length" class="flex-grow-1 d-flex">
					<div class="empty-box p-5 text-center">
						<span class="text-muted">{{ $t("main.history.empty") }}</span>
					</div>
				</div>

				<section
					v-for="group in visibleGroups"
					:key="group"
					class="history-tile mb-4"
					:data-testid="`history-section-${group}`"
				>
					<h3
						class="fw-normal my-0 d-flex gap-3 flex-wrap align-items-baseline overflow-hidden history-tile-title"
					>
						<span class="d-block no-wrap text-truncate">
							{{ $t(`main.history.group.${group}`) }}
						</span>
						<small class="d-block no-wrap text-truncate">
							{{ groupTotalLabel(group) }}
						</small>
					</h3>
					<GroupChart
						v-if="displayFrom && displayTo && displayPeriod"
						:group="group"
						:color="groupColor(group)"
						:series="displaySeries(group)"
						:overlay="
							group === 'pv' && displayPeriod === 'day'
								? seriesByGroup['forecast']
								: []
						"
						:overlayColor="groupColor('forecast')"
						:overlayLabel="$t('main.history.group.forecast')"
						:showOverlay="
							group === 'pv' &&
							displayPeriod === 'day' &&
							hasForecast &&
							(focusedEntity[group] ?? null) === null
						"
						:focusedEntity="focusedEntity[group] ?? null"
						:period="displayPeriod"
						:from="displayFrom"
						:to="displayTo"
					/>
					<ul
						v-if="hasEntityLegend(group) || hasForecastLegend(group)"
						class="entity-legend p-0 mt-4 mb-0 d-flex flex-wrap column-gap-4 row-gap-2"
					>
						<li
							v-for="legend in entityLegends(group)"
							:key="legend.entityIndex"
							class="entity-legend-item d-flex align-items-baseline gap-2 no-wrap"
							:class="{
								'entity-legend-item--dim': isDimmed(group, legend.entityIndex),
							}"
							role="button"
							tabindex="0"
							@click="toggleFocus(group, legend.entityIndex)"
							@keydown.enter.prevent="toggleFocus(group, legend.entityIndex)"
							@keydown.space.prevent="toggleFocus(group, legend.entityIndex)"
						>
							<span
								class="entity-legend-dot align-self-center"
								:style="{ backgroundColor: legend.color }"
							></span>
							<span class="text-nowrap">{{ legend.label }}</span>
							<span class="text-muted text-nowrap">{{ legend.value }}</span>
						</li>
						<li
							v-if="hasForecastLegend(group)"
							class="entity-legend-item entity-legend-item--static d-flex align-items-baseline gap-2 no-wrap"
							:class="{
								'entity-legend-item--dim': (focusedEntity[group] ?? null) !== null,
							}"
						>
							<span
								class="entity-legend-dot entity-legend-dot--line align-self-center"
								:style="{ backgroundColor: groupColor('forecast') }"
							></span>
							<span class="text-nowrap">
								{{ $t("main.history.group.forecast") }}
							</span>
							<span class="text-muted text-nowrap">
								{{ forecastTotalLabel }}
							</span>
						</li>
					</ul>
				</section>
				<p v-if="visibleGroups.length" class="text-end mt-3 mb-0">
					<a
						:href="csvLink"
						download
						class="text-muted small history-csv-link"
						data-testid="history-csv-download"
					>
						{{ $t("main.history.downloadCsv") }}
					</a>
				</p>
			</main>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Header from "../components/Top/Header.vue";
import PeriodSelector from "../components/Sessions/PeriodSelector.vue";
import DateNavigator from "../components/Sessions/DateNavigator.vue";
import PeriodHeader from "../components/Sessions/PeriodHeader.vue";
import GroupChart, {
	type HistorySeries,
	alphaColor,
	stepAlpha,
} from "../components/History/GroupChart.vue";
import type { Legend } from "../components/Sessions/types";
import { PERIODS } from "../components/Sessions/types";
import { GROUP_ORDER, groupColor } from "../components/History/groups";
import colors from "../colors";
import formatter, { POWER_UNIT } from "../mixins/formatter";
import api from "../api";
import store from "../store";

const HISTORY_PERIODS = [PERIODS.DAY, PERIODS.MONTH, PERIODS.YEAR];

function daysInMonth(year: number, month: number) {
	return new Date(year, month, 0).getDate();
}

export default defineComponent({
	name: "History",
	components: {
		TopHeader: Header,
		PeriodSelector,
		DateNavigator,
		PeriodHeader,
		GroupChart,
	},
	mixins: [formatter],
	props: {
		day: { type: Number, default: undefined },
		month: { type: Number, default: undefined },
		year: { type: Number, default: undefined },
		period: { type: String as PropType<PERIODS>, default: undefined },
	},
	data() {
		return {
			rawSeries: [] as HistorySeries[],
			displayFrom: null as Date | null,
			displayTo: null as Date | null,
			displayPeriod: null as PERIODS | null,
			loading: true,
			interval: null as ReturnType<typeof setInterval> | null,
			startDate: new Date(2020, 0, 1),
			focusedEntity: {} as Record<string, number | null>,
		};
	},
	head() {
		return { title: this.$t("main.history.title") };
	},
	computed: {
		effectivePeriod(): PERIODS {
			return this.period && HISTORY_PERIODS.includes(this.period)
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
			return HISTORY_PERIODS.map((value) => ({
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
				case PERIODS.MONTH:
				default:
					return new Date(this.effectiveYear, this.effectiveMonth - 1, 1);
			}
		},
		to(): Date {
			switch (this.effectivePeriod) {
				case PERIODS.DAY: {
					const d = new Date(this.from);
					d.setDate(d.getDate() + 1);
					return d;
				}
				case PERIODS.YEAR:
					return new Date(this.effectiveYear + 1, 0, 1);
				case PERIODS.MONTH:
				default:
					return new Date(this.effectiveYear, this.effectiveMonth, 1);
			}
		},
		aggregate(): string {
			switch (this.effectivePeriod) {
				case PERIODS.DAY:
					return "15m";
				case PERIODS.YEAR:
					return "month";
				case PERIODS.MONTH:
				default:
					return "day";
			}
		},
		fetchKey(): string {
			return `${this.aggregate}|${this.from.getTime()}|${this.to.getTime()}`;
		},
		seriesByGroup(): Record<string, HistorySeries[]> {
			const titles = store.deviceTitles.value;
			const map: Record<string, HistorySeries[]> = {};
			for (const s of this.rawSeries) {
				if (!s.group) continue;
				if (!map[s.group]) map[s.group] = [];
				map[s.group]!.push({ ...s, name: titles[s.name] || s.name });
			}
			return map;
		},
		visibleGroups(): string[] {
			return GROUP_ORDER.filter((g) => {
				// Consumption uses `home` as the source of truth, so the section
				// shows up whenever home has data, even without explicit meters.
				if (g === "meter") {
					const home = this.seriesByGroup["home"];
					if (
						home?.some((s) =>
							s.data.some((slot) => slot.energy !== 0 || slot.returnEnergy !== 0)
						)
					) {
						return true;
					}
				}
				const list = this.seriesByGroup[g];
				if (!list?.length) return false;
				return list.some((s) =>
					s.data.some((slot) => slot.energy !== 0 || slot.returnEnergy !== 0)
				);
			});
		},
		// Series shown in the chart. For the consumption (`meter`) group we append
		// a virtual "Other consumers" series = home.net − sum(meter entities).
		// Inactive loadpoints / meters (all-zero data in the selected period) are
		// dropped from the displayed list; `paletteIndex` carries each entity's
		// original position so its color stays stable when navigating periods.
		displaySeries(): (group: string) => HistorySeries[] {
			const hasEnergy = (s: HistorySeries) =>
				s.data.some((slot) => slot.energy !== 0 || slot.returnEnergy !== 0);
			return (group: string): HistorySeries[] => {
				if (group === "loadpoint") {
					const list = this.seriesByGroup["loadpoint"] || [];
					return list.map((s, i) => ({ ...s, paletteIndex: i })).filter(hasEnergy);
				}
				if (group !== "meter") return this.seriesByGroup[group] || [];
				const meters = this.seriesByGroup["meter"] || [];
				const home = (this.seriesByGroup["home"] || [])[0];
				const activeMeters = meters
					.map((s, i) => ({ ...s, paletteIndex: i }))
					.filter(hasEnergy);
				if (!home) return activeMeters;
				const meterTotals = new Map<string, number>();
				// Net per slot is computed from all meters (incl. inactive ones), so
				// dropping inactive entries from the display doesn't shift the
				// "Other consumers" delta.
				for (const s of meters) {
					for (const slot of s.data) {
						const net = slot.energy - slot.returnEnergy;
						meterTotals.set(slot.start, (meterTotals.get(slot.start) || 0) + net);
					}
				}
				const other: HistorySeries = {
					name: this.$t("main.history.otherConsumers") as string,
					group: "meter",
					virtual: true,
					// Use meters.length as a stable paletteIndex that can never
					// collide with real meters (0..meters.length-1).
					paletteIndex: meters.length,
					data: home.data.map((slot) => {
						const homeNet = slot.energy - slot.returnEnergy;
						const v = Math.max(0, homeNet - (meterTotals.get(slot.start) || 0));
						return { start: slot.start, end: slot.end, energy: v, returnEnergy: 0 };
					}),
				};
				if (!hasEnergy(other)) return activeMeters;
				// First entry in the array = bottom of the stack, so "Other consumers"
				// always sits underneath the explicit meters.
				return [other, ...activeMeters];
			};
		},
		hasForecast(): boolean {
			const list = this.seriesByGroup["forecast"];
			if (!list?.length) return false;
			return list.some((s) =>
				s.data.some((slot) => slot.energy !== 0 || slot.returnEnergy !== 0)
			);
		},
		forecastTotalLabel(): string {
			const list = this.seriesByGroup["forecast"] || [];
			let sum = 0;
			for (const s of list) {
				for (const slot of s.data) sum += slot.energy - slot.returnEnergy;
			}
			return this.fmtWh(Math.abs(sum) * 1000, POWER_UNIT.AUTO);
		},
		csvLink(): string {
			const params = new URLSearchParams({
				format: "csv",
				lang: this.$i18n?.locale,
				from: this.from.toISOString(),
				to: this.to.toISOString(),
				aggregate: this.aggregate,
			});
			return `./api/history/energy?${params.toString()}`;
		},
	},
	watch: {
		fetchKey() {
			this.fetchData();
		},
		rawSeries() {
			// Drop focused entries whose paletteIndex no longer matches any entity
			// in the new data (e.g. a loadpoint was filtered out after a period
			// switch).
			const next: Record<string, number | null> = {};
			for (const [group, focused] of Object.entries(this.focusedEntity)) {
				if (focused == null) {
					next[group] = null;
					continue;
				}
				const list = this.displaySeries(group);
				const stillPresent = list.some((s, i) => (s.paletteIndex ?? i) === focused);
				next[group] = stillPresent ? focused : null;
			}
			this.focusedEntity = next;
		},
	},
	mounted() {
		this.fetchData();
		this.interval = setInterval(() => this.fetchData(), 15 * 60 * 1e3);
	},
	unmounted() {
		if (this.interval) clearInterval(this.interval);
	},
	methods: {
		groupColor,
		hasEntityLegend(group: string): boolean {
			const list = this.displaySeries(group);
			if (!list.length) return false;
			if (group === "loadpoint" || group === "meter") return true;
			if (group === "pv" || group === "battery") return list.length > 1;
			return false;
		},
		hasForecastLegend(group: string): boolean {
			return group === "pv" && this.hasForecast && this.effectivePeriod === PERIODS.DAY;
		},
		entityLegends(group: string): (Legend & { entityIndex: number })[] {
			const list = this.displaySeries(group);
			const baseColor = groupColor(group);
			const n = list.length;
			const colorFor = (i: number, s: HistorySeries) => {
				if (s.virtual) return colors.muted || baseColor;
				if (group === "loadpoint" || group === "meter") {
					const idx = s.paletteIndex ?? i;
					return colors.palette[idx % colors.palette.length] || baseColor;
				}
				return alphaColor(baseColor, stepAlpha(i, Math.max(n, 1)));
			};
			return list.map((s, i) => {
				let sum = 0;
				for (const slot of s.data) sum += slot.energy - slot.returnEnergy;
				const watts = Math.abs(sum) * 1000;
				return {
					// Use stable paletteIndex as the focus identifier so that the
					// selected entity keeps its identity across period navigations.
					entityIndex: s.paletteIndex ?? i,
					label: s.name,
					color: colorFor(i, s),
					value: this.fmtWh(watts, POWER_UNIT.AUTO),
				};
			});
		},
		toggleFocus(group: string, i: number) {
			const current = this.focusedEntity[group] ?? null;
			this.focusedEntity = {
				...this.focusedEntity,
				[group]: current === i ? null : i,
			};
		},
		isDimmed(group: string, i: number): boolean {
			const focused = this.focusedEntity[group] ?? null;
			return focused !== null && focused !== i;
		},
		groupTotalLabel(group: string): string {
			// Consumption total comes from `home` (overall consumption),
			// not the sum of explicit meter entities.
			const list =
				group === "meter"
					? this.seriesByGroup["home"] || []
					: this.seriesByGroup[group] || [];
			let sumEnergy = 0;
			let sumReturnEnergy = 0;
			for (const s of list) {
				for (const slot of s.data) {
					sumEnergy += slot.energy;
					sumReturnEnergy += slot.returnEnergy;
				}
			}
			const fmt = (v: number) => this.fmtWh(v * 1000, POWER_UNIT.KW);
			const directionKey = `main.history.direction.${group}`;
			const energyKey = `${directionKey}.energy`;
			const returnEnergyKey = `${directionKey}.returnEnergy`;
			const energyLabel = this.$t(energyKey);
			const returnEnergyLabel = this.$t(returnEnergyKey);
			const hasDirectionLabels =
				energyLabel !== energyKey && returnEnergyLabel !== returnEnergyKey;
			if (sumEnergy > 0 && sumReturnEnergy > 0 && hasDirectionLabels) {
				return `${fmt(sumEnergy)} ${energyLabel} · ${fmt(sumReturnEnergy)} ${returnEnergyLabel}`;
			}
			return fmt(Math.abs(sumEnergy - sumReturnEnergy) || sumEnergy + sumReturnEnergy);
		},
		async fetchData() {
			this.loading = true;
			const requestFrom = this.from;
			const requestTo = this.to;
			const requestPeriod = this.effectivePeriod;
			try {
				const res = await api.get("history/energy", {
					params: {
						from: requestFrom.toISOString(),
						to: requestTo.toISOString(),
						aggregate: this.aggregate,
						grouped: false,
					},
				});
				this.rawSeries = res.data || [];
				this.displayFrom = requestFrom;
				this.displayTo = requestTo;
				this.displayPeriod = requestPeriod;
			} catch (e) {
				console.error("Failed to load energy history", e);
				this.rawSeries = [];
			} finally {
				this.loading = false;
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
.history-tile {
	background: var(--evcc-box);
	padding: 1.25rem 1rem 1.75rem;
}
.empty-container {
	min-height: calc(100dvh - var(--bottom-space));
}
.empty-box {
	background-color: var(--evcc-box);
	margin: auto;
	border-radius: 2rem;
	max-width: 480px;
}
.history-tile-skeleton {
	min-height: 270px;
	animation: history-skeleton-pulse 1.6s ease-in-out infinite;
}
@keyframes history-skeleton-pulse {
	0%,
	100% {
		opacity: 1;
	}
	50% {
		opacity: 0.6;
	}
}
@media (prefers-reduced-motion: reduce) {
	.history-tile-skeleton {
		animation: none;
		opacity: 0.8;
	}
}
.entity-legend {
	list-style: none;
}
.entity-legend-item {
	cursor: pointer;
	transition: opacity 150ms;
	user-select: none;
}
.entity-legend-item--dim {
	opacity: 0.35;
}
.entity-legend-dot {
	display: inline-block;
	width: 1rem;
	height: 1rem;
	border-radius: 50%;
	flex-shrink: 0;
}
.history-csv-link {
	text-decoration: none;
}
.history-csv-link:hover,
.history-csv-link:focus {
	color: var(--evcc-default-text);
	text-decoration: underline;
}
.entity-legend-dot--line {
	height: 2px;
	border-radius: 1px;
}
.entity-legend-item--static {
	cursor: default;
}
.history-tile-title {
	margin: 0 0 0.5rem;
}
@media (max-width: 575.98px) {
	/* edge-to-edge on mobile: cancel the container's px-4 padding */
	.history-tile {
		margin-left: -1.5rem;
		margin-right: -1.5rem;
		border-radius: 0;
	}
}
@media (min-width: 576px) {
	.history-tile {
		border-radius: 1rem;
		padding: 1.5rem 1.5rem 2rem;
	}
}
</style>
