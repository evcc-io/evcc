<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader :title="$t('sessions.title')" :notifications="notifications" />
		<div class="row">
			<main class="col-12">
				<div class="header-outer sticky-top">
					<div class="container px-4">
						<div
							class="row py-3 py-sm-3 d-flex flex-column flex-sm-row gap-3 gap-lg-0 mb-lg-2"
						>
							<div class="col-lg-5 d-flex mb-lg-0">
								<PeriodSelector
									:period="period"
									:periodOptions="periodOptions"
									@update:period="changePeriod"
								/>
							</div>
							<div v-if="showDateNavigator" class="col-lg-6 offset-lg-1">
								<DateNavigator
									:month="month"
									:year="year"
									:startDate="startDate"
									:showMonth="showMonthNavigation"
									:showYear="showYearNavigation"
									@update-date="updateDate"
								/>
							</div>
						</div>
					</div>
				</div>

				<div class="d-flex gap-3 mb-5 justify-content-between flex-wrap pt-1">
					<IconSelectGroup>
						<template v-for="largeScreen in [true, false]">
							<IconSelectItem
								v-for="{ value, label, disabled, active } in typeOptions"
								:key="value + largeScreen"
								:label="largeScreen ? label : undefined"
								:class="{
									'd-none d-lg-block': largeScreen,
									'd-block d-lg-none': !largeScreen,
								}"
								:disabled="disabled"
								:active="active"
								@click="updateType(value)"
							>
								<component :is="typeIcons[value]"></component>
							</IconSelectItem>
						</template>
					</IconSelectGroup>
					<IconSelectGroup>
						<template v-for="largeScreen in [true, false]">
							<IconSelectItem
								v-for="group in Object.values(groups)"
								:key="group + largeScreen"
								:active="selectedGroup === group"
								:label="
									largeScreen
										? $t(`sessions.groupBy.${group.toLowerCase()}`)
										: undefined
								"
								:class="{
									'd-none d-lg-block': largeScreen,
									'd-block d-lg-none': !largeScreen,
								}"
								@click="updateGroup(group)"
							>
								<component :is="groupIcons[group]"></component>
							</IconSelectItem>
						</template>
					</IconSelectGroup>
				</div>

				<h3
					class="fw-normal my-0 d-flex gap-3 flex-wrap d-flex align-items-baseline overflow-hidden"
				>
					<span v-if="historyTitle" class="d-block no-wrap text-truncate">
						{{ historyTitle }}
					</span>
					<small class="d-block no-wrap text-truncate">{{ historySubTitle }}</small>
				</h3>
				<EnergyHistoryChart
					v-if="activeType === types.SOLAR"
					class="mb-5"
					:sessions="currentSessions"
					:color-mappings="colorMappings"
					:group-by="selectedGroup"
					:period="period"
				/>
				<CostHistoryChart
					v-else
					class="mb-5"
					:sessions="currentTypeSessions"
					:color-mappings="colorMappings"
					:group-by="selectedGroup"
					:cost-type="activeType"
					:currency="currency"
					:period="period"
					:suggested-max-avg-cost="suggestedMaxAvgCost"
					:suggested-max-cost="suggestedMaxCost"
				/>
				<div v-if="showExtraCharts" class="row align-items-start">
					<div class="col-12 col-lg-6 mb-5">
						<h3 class="fw-normal my-4">{{ firstExtraTitle }}</h3>
						<div v-if="activeType === types.SOLAR">
							<SolarYearChart
								v-if="showSolarYearChart"
								:period="period"
								:sessions="currentSessions"
							/>
							<SolarGroupedChart
								v-else
								:sessions="currentSessions"
								:color-mappings="colorMappings"
								:group-by="selectedGroupWithoutNone"
							/>
						</div>
						<AvgCostGroupedChart
							v-else
							:sessions="currentTypeSessions"
							:color-mappings="colorMappings"
							:suggested-max-price="suggestedMaxAvgCost"
							:group-by="selectedGroupWithoutNone"
							:cost-type="activeType"
							:currency="currency"
						/>
					</div>
					<div class="col-12 col-lg-6 mb-5">
						<h3 class="fw-normal my-4">{{ secondExtraTitle }}</h3>
						<EnergyGroupedChart
							v-if="activeType === types.SOLAR"
							:sessions="currentSessions"
							:color-mappings="colorMappings"
							:group-by="selectedGroupWithoutNone"
						/>
						<CostGroupedChart
							v-else
							:sessions="currentTypeSessions"
							:color-mappings="colorMappings"
							:group-by="selectedGroupWithoutNone"
							:cost-type="activeType"
							:currency="currency"
						/>
					</div>
				</div>

				<SessionTable
					v-if="showTable"
					:sessions="currentSessions"
					:vehicleFilter="vehicleFilter"
					:loadpointFilter="loadpointFilter"
					:currency="currency"
					@show-session="showDetails"
				/>
				<div class="d-flex gap-2 mt-1 mb-5">
					<a class="btn btn-outline-secondary" tabindex="0" :href="csvLink" download>
						{{ csvLinkLabel }}
					</a>
					<button
						v-if="!showTable"
						class="btn btn-link text-muted"
						@click="changePeriod(periods.MONTH)"
					>
						{{ $t("sessions.showIndividualEntries") }}
					</button>
				</div>
			</main>
			<SessionDetailsModal
				:session="selectedSession"
				:vehicles="vehicleList"
				:loadpoints="loadpointList"
				:currency="currency"
				@session-changed="loadSessions"
			/>
		</div>
	</div>
</template>

<script lang="ts">
import Modal from "bootstrap/js/dist/modal";
import "@h2d2/shopicons/es/regular/cablecharge";
import "@h2d2/shopicons/es/regular/car3";
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/sun";
import formatter, { POWER_UNIT } from "../mixins/formatter";
import api from "../api";
import store from "../store";
import SessionDetailsModal from "../components/Sessions/SessionDetailsModal.vue";
import SessionTable from "../components/Sessions/SessionTable.vue";
import EnergyHistoryChart from "../components/Sessions/EnergyHistoryChart.vue";
import EnergyGroupedChart from "../components/Sessions/EnergyGroupedChart.vue";
import SolarGroupedChart from "../components/Sessions/SolarGroupedChart.vue";
import SolarYearChart from "../components/Sessions/SolarYearChart.vue";
import CostHistoryChart from "../components/Sessions/CostHistoryChart.vue";
import CostGroupedChart from "../components/Sessions/CostGroupedChart.vue";
import AvgCostGroupedChart from "../components/Sessions/AvgCostGroupedChart.vue";
import Header from "../components/Top/Header.vue";
import IconSelectGroup from "../components/Helper/IconSelectGroup.vue";
import IconSelectItem from "../components/Helper/IconSelectItem.vue";
import SelectGroup from "../components/Helper/SelectGroup.vue";
import CustomSelect from "../components/Helper/CustomSelect.vue";
import colors from "../colors";
import settings from "../settings";
import PeriodSelector from "../components/Sessions/PeriodSelector.vue";
import DateNavigator from "../components/Sessions/DateNavigator.vue";
import DynamicPriceIcon from "../components/MaterialIcon/DynamicPrice.vue";
import TotalIcon from "../components/MaterialIcon/Total.vue";
import { TYPES, GROUPS, PERIODS, type Session } from "../components/Sessions/types";
import { defineComponent, type PropType } from "vue";
import { CURRENCY, type Notification } from "@/types/evcc";

export default defineComponent({
	name: "Sessions",
	components: {
		SessionDetailsModal,
		SessionTable,
		TopHeader: Header,
		EnergyHistoryChart,
		EnergyGroupedChart,
		IconSelectGroup,
		IconSelectItem,
		SelectGroup,
		CustomSelect,
		SolarGroupedChart,
		SolarYearChart,
		PeriodSelector,
		DateNavigator,
		DynamicPriceIcon,
		CostHistoryChart,
		CostGroupedChart,
		AvgCostGroupedChart,
	},
	mixins: [formatter],
	props: {
		notifications: Array as PropType<Notification[]>,
		month: { type: Number, default: () => new Date().getMonth() + 1 },
		year: { type: Number, default: () => new Date().getFullYear() },
		period: { type: String as PropType<PERIODS>, default: PERIODS.MONTH },
		loadpointFilter: { type: String, default: "" },
		vehicleFilter: { type: String, default: "" },
		offline: Boolean,
	},
	data() {
		return {
			sessions: [] as Session[],
			selectedType: (settings.sessionsType || TYPES.SOLAR) as TYPES,
			selectedGroup: (settings.sessionsGroup || GROUPS.NONE) as GROUPS,
			selectedSessionId: undefined as number | undefined,
			periods: PERIODS,
			types: TYPES,
			groups: GROUPS,
		};
	},
	head() {
		return { title: this.$t("sessions.title") };
	},
	computed: {
		selectedGroupWithoutNone() {
			return this.selectedGroup !== this.groups.NONE ? this.selectedGroup : undefined;
		},
		currency() {
			return store.state.currency || CURRENCY.EUR;
		},
		energyTitle() {
			return this.$t("sessions.chartTitle.energy");
		},
		historyTitle() {
			return this.activeType === TYPES.SOLAR ? this.energyTitle : this.costTitle;
		},
		historySubTitle() {
			if (this.activeType === TYPES.SOLAR) {
				return this.energySubTitle;
			}
			return this.costSubTitle;
		},
		firstExtraTitle() {
			if (this.activeType === TYPES.SOLAR) {
				return this.solarTitle;
			}
			return this.avgCostTitle;
		},
		secondExtraTitle() {
			if (this.activeType === TYPES.SOLAR) {
				return this.energyGroupedTitle;
			}
			return this.costGroupedTitle;
		},
		solarPercentageFmt() {
			return this.fmtPercentage(
				this.totalEnergy > 0 ? (100 / this.totalEnergy) * this.selfEnergy : 0
			);
		},
		energySumFmt() {
			return this.fmtWh(this.totalEnergy * 1e3, POWER_UNIT.AUTO);
		},
		energySubTitle() {
			const total = this.$t("sessions.chartTitle.energySubTotal", {
				value: this.energySumFmt,
			});
			const solar = this.$t("sessions.chartTitle.energySubSolar", {
				value: this.solarPercentageFmt,
			});
			return `${total} ・ ${solar}`;
		},
		solarTitle() {
			return this.selectedGroup === GROUPS.NONE
				? this.$t("sessions.chartTitle.solar")
				: this.$t("sessions.chartTitle.solarByGroup", { byGroup: this.byGroupTitle });
		},
		byGroupTitle() {
			if (this.selectedGroup === GROUPS.LOADPOINT) {
				return this.$t("sessions.chartTitle.byGroupLoadpoint");
			} else if (this.selectedGroup === GROUPS.VEHICLE) {
				return this.$t("sessions.chartTitle.byGroupVehicle");
			}
			return "";
		},
		energyGroupedTitle() {
			if (this.selectedGroup === GROUPS.NONE) {
				return this.$t("sessions.chartTitle.energyGrouped");
			}
			return this.$t("sessions.chartTitle.energyGroupedByGroup", {
				byGroup: this.byGroupTitle,
			});
		},
		avgCostTitle() {
			const type = this.activeType === TYPES.PRICE ? "Price" : "Co2";
			return this.$t(`sessions.chartTitle.avg${type}ByGroup`, {
				byGroup: this.byGroupTitle,
			});
		},
		costGroupedTitle() {
			const type = this.activeType === TYPES.PRICE ? "Price" : "Co2";
			return this.$t(`sessions.chartTitle.grouped${type}ByGroup`, {
				byGroup: this.byGroupTitle,
			});
		},
		periodOptions() {
			return Object.entries(PERIODS).map(([key, value]) => ({
				name: this.$t(`sessions.period.${key.toLowerCase()}`),
				value,
			}));
		},
		typeOptions() {
			const options = Object.values(TYPES).map((value) => {
				const disabled =
					(value === TYPES.PRICE && !this.typePriceAvailable) ||
					(value === TYPES.CO2 && !this.typeCo2Available);
				const active = this.activeType === value;
				const label = this.$t(`sessions.type.${value}`);
				return { label, value, disabled, active };
			});
			return options;
		},
		totalEnergy() {
			return this.currentSessions.reduce((acc, session) => acc + session.chargedEnergy, 0);
		},
		selfEnergy() {
			return this.currentSessions.reduce(
				(acc, session) => acc + (session.chargedEnergy / 100) * session.solarPercentage,
				0
			);
		},
		currentSessionsWithPrice() {
			return this.currentSessions.filter((s) => s.price !== null);
		},
		currentTypeSessions() {
			if (this.activeType === TYPES.PRICE) {
				return this.currentSessionsWithPrice;
			} else if (this.activeType === TYPES.CO2) {
				return this.currentSessionsWithCo2;
			} else {
				return this.currentSessions;
			}
		},
		totalPrice() {
			return this.currentSessionsWithPrice.reduce((acc, s) => acc + (s.price ?? 0), 0);
		},
		pricePerKWh() {
			const list = this.currentSessionsWithPrice;
			const energy = list.reduce((acc, s) => acc + s.chargedEnergy, 0);
			return energy ? this.totalPrice / energy : 0;
		},
		currentSessionsWithCo2() {
			return this.currentSessions.filter((s) => s.co2PerKWh !== null);
		},
		totalCo2() {
			const list = this.currentSessionsWithCo2;
			return list.reduce((acc, s) => acc + (s.co2PerKWh ?? 0) * s.chargedEnergy, 0);
		},
		co2PerKWh() {
			const list = this.currentSessionsWithCo2;
			const energy = list.reduce((acc, s) => acc + s.chargedEnergy, 0);
			return energy ? this.totalCo2 / energy : 0;
		},
		costTitle() {
			const type = this.activeType === TYPES.PRICE ? "Price" : "Co2";
			return this.$t(`sessions.chartTitle.history${type}`);
		},
		avgCostFmt() {
			return this.activeType === TYPES.PRICE
				? this.fmtPricePerKWh(this.pricePerKWh, this.currency)
				: this.fmtCo2Medium(this.co2PerKWh);
		},
		costSubTitle() {
			const type = this.activeType === TYPES.PRICE ? "Price" : "Co2";
			const value =
				this.activeType === TYPES.PRICE
					? this.fmtMoney(this.totalPrice, this.currency, true, true)
					: this.fmtGrams(this.totalCo2);
			const total = this.$t(`sessions.chartTitle.history${type}Sub`, { value });
			return `${total} ・ ⌀ ${this.avgCostFmt}`;
		},
		activeType() {
			if (this.selectedType === TYPES.PRICE && this.typePriceAvailable) {
				return TYPES.PRICE;
			} else if (this.selectedType === TYPES.CO2 && this.typeCo2Available) {
				return TYPES.CO2;
			}
			return TYPES.SOLAR;
		},
		showCostCharts() {
			return this.typePriceAvailable || this.typeCo2Available;
		},
		typePriceAvailable() {
			return this.currentSessionsWithPrice.length > 0;
		},
		typeCo2Available() {
			return this.currentSessionsWithCo2.length > 0;
		},
		startDate() {
			return new Date(this.sessions[0]?.created || Date.now());
		},
		sessionsWithDefaults() {
			return this.sessions.map((session) => {
				const loadpoint = session.loadpoint || this.$t("main.loadpoint.fallbackName");
				const vehicle = session.vehicle || this.$t("main.vehicle.unknown");
				return { ...session, loadpoint, vehicle };
			});
		},
		currentSessions() {
			return this.sessionsWithDefaults.filter((session) => {
				const date = new Date(session.created);
				switch (this.period) {
					case PERIODS.MONTH:
						return (
							date.getFullYear() === this.year && date.getMonth() + 1 === this.month
						);
					case PERIODS.YEAR:
						return date.getFullYear() === this.year;
					case PERIODS.TOTAL:
						return true;
					default:
						return false;
				}
			});
		},
		vehicleList() {
			const vehicles = store.state.vehicles || {};
			return Object.entries(vehicles).map(([name, vehicle]) => ({ ...vehicle, name }));
		},
		loadpointList() {
			const loadpoints = store.state.loadpoints || [];
			return loadpoints.map(({ title }) => title);
		},
		selectedSession() {
			return this.sessions.find((s) => s.id == this.selectedSessionId);
		},
		monthName() {
			const date = new Date();
			date.setMonth(this.month - 1, 1);
			return this.fmtMonth(date, false);
		},
		csvLinkLabel() {
			if (this.period === PERIODS.MONTH) {
				const date = new Date();
				date.setMonth(this.month - 1, 1);
				date.setFullYear(this.year);
				const period = this.fmtMonthYear(date);
				return this.$t("sessions.csvPeriod", { period });
			} else if (this.period === PERIODS.YEAR) {
				const period = this.year;
				return this.$t("sessions.csvPeriod", { period });
			} else {
				return this.$t("sessions.csvTotal");
			}
		},
		csvLink() {
			if (this.period === PERIODS.MONTH) {
				return this.csvHrefLink(this.year, this.month);
			} else if (this.period === PERIODS.YEAR) {
				return this.csvHrefLink(this.year);
			}
			return this.csvHrefLink();
		},
		colorMappings() {
			const lastThreeMonths = new Date();
			lastThreeMonths.setMonth(lastThreeMonths.getMonth() - 3);

			// Aggregate energy to get sorted list of loadpoints/vehicles for coloring
			const aggregateEnergy = (group: Exclude<GROUPS, GROUPS.NONE>) => {
				return this.sessionsWithDefaults.reduce((acc: Record<string, number>, session) => {
					if (new Date(session.created) >= lastThreeMonths) {
						const key = session[group];
						acc[key] = (acc[key] || 0) + session.chargedEnergy;
					}
					return acc;
				}, {});
			};

			// Assign colors based on energy usage
			const assignColors = (
				energyAggregation: Record<string, number>,
				colorType: Exclude<GROUPS, GROUPS.NONE>
			) => {
				const result: Record<string, string> = {};
				let colorIndex = 0;

				// Assign colors by used energy in the last three months
				const sortedEntries = Object.entries(energyAggregation).sort((a, b) => b[1] - a[1]);
				sortedEntries.forEach(([key]) => {
					if (key && !result[key]) {
						result[key] = colors.palette[colorIndex % colors.palette.length] || "";
						colorIndex++;
					}
				});

				// Assign colors to remaining entries
				this.sessionsWithDefaults.forEach((session) => {
					const key = session[colorType];
					if (key && !result[key]) {
						result[key] = colors.palette[colorIndex % colors.palette.length] || "";
						colorIndex++;
					}
				});

				return result;
			};

			const loadpointEnergy = aggregateEnergy(GROUPS.LOADPOINT);
			const loadpointColors = assignColors(loadpointEnergy, GROUPS.LOADPOINT);

			const vehicleEnergy = aggregateEnergy(GROUPS.VEHICLE);
			const vehicleColors = assignColors(vehicleEnergy, GROUPS.VEHICLE);

			const solar = { self: colors.self, grid: colors.grid };
			const cost = { price: colors.price, co2: colors.co2 };

			return {
				loadpoint: loadpointColors,
				vehicle: vehicleColors,
				solar,
				cost,
			};
		},
		groupIcons() {
			return {
				[GROUPS.NONE]: TotalIcon,
				[GROUPS.LOADPOINT]: "shopicon-regular-cablecharge",
				[GROUPS.VEHICLE]: "shopicon-regular-car3",
			};
		},
		typeIcons() {
			return {
				[TYPES.SOLAR]: "shopicon-regular-sun",
				[TYPES.PRICE]: DynamicPriceIcon,
				[TYPES.CO2]: "shopicon-regular-eco1",
			};
		},
		costTypeIcons() {
			return {
				[TYPES.PRICE]: DynamicPriceIcon,
				[TYPES.CO2]: "shopicon-regular-eco1",
			};
		},
		showTable() {
			return this.period === PERIODS.MONTH;
		},
		showMonthNavigation() {
			return this.period === PERIODS.MONTH;
		},
		showYearNavigation() {
			return [PERIODS.MONTH, PERIODS.YEAR].includes(this.period);
		},
		showDateNavigator() {
			return this.showMonthNavigation || this.showYearNavigation;
		},
		groupEntriesAvailable() {
			if (this.selectedGroup === GROUPS.NONE || !this.currentSessions.length) return false;
			return (
				new Set(
					this.currentSessions.map(
						(s) => s[this.selectedGroup as Exclude<GROUPS, GROUPS.NONE>]
					)
				).size > 1
			);
		},
		showSolarYearChart() {
			return this.period !== PERIODS.MONTH && this.selectedGroup === GROUPS.NONE;
		},
		showExtraCharts() {
			const hasMultipleEntries =
				new Set(
					this.currentTypeSessions.map(
						(s) => s[this.selectedGroup as Exclude<GROUPS, GROUPS.NONE>]
					)
				).size > 1;
			const isGrouped = [GROUPS.LOADPOINT, GROUPS.VEHICLE].includes(this.selectedGroup);
			const isSolar = this.activeType === TYPES.SOLAR;
			const isNotMonth = this.period !== PERIODS.MONTH;

			return (isGrouped && hasMultipleEntries) || (isSolar && isNotMonth && !isGrouped);
		},
		suggestedMaxAvgPrice() {
			// returns the 98th percentile of avg prices for all sessions
			const sessionsWithPrice = this.sessions.filter((s) => s.pricePerKWh !== null);
			const prices = sessionsWithPrice.map((s) => s.pricePerKWh ?? 0);
			return this.percentile(prices, 98) ?? 0;
		},
		suggestedMaxAvgCo2() {
			// returns the 98th percentile of avg co2 emissions for all sessions
			const sessionsWithCo2 = this.sessions.filter((s) => s.co2PerKWh !== null);
			const co2 = sessionsWithCo2.map((s) => s.co2PerKWh ?? 0);
			return this.percentile(co2, 98) ?? 0;
		},
		suggestedMaxAvgCost() {
			return this.activeType === TYPES.PRICE
				? this.suggestedMaxAvgPrice
				: this.suggestedMaxAvgCo2;
		},
		suggestedMaxCo2() {
			// returns the 98th percentile of total co2 emissions by time period
			const sessionsWithCo2 = this.sessions.filter((s) => s.co2PerKWh !== null);
			const co2Map = sessionsWithCo2.reduce((acc: Record<string, number>, s) => {
				const key = this.dateToPeriodKey(new Date(s.created));
				acc[key] = (acc[key] || 0) + (s.co2PerKWh ?? 0) * s.chargedEnergy;
				return acc;
			}, {});
			return Math.max(this.percentile(Object.values(co2Map), 98) ?? 0, 5); // 5kg default
		},
		suggestedMaxPrice() {
			// returns the 98th percentile of total price by time period
			const sessionsWithPrice = this.sessions.filter((s) => s.price !== null);
			const priceMap = sessionsWithPrice.reduce((acc: Record<string, number>, s) => {
				const key = this.dateToPeriodKey(new Date(s.created));
				acc[key] = (acc[key] || 0) + (s.price || 0);
				return acc;
			}, {});
			return Math.max(this.percentile(Object.values(priceMap), 98) ?? 0, 1); // 1 CURRENCY default
		},
		suggestedMaxCost() {
			return this.activeType === TYPES.PRICE ? this.suggestedMaxPrice : this.suggestedMaxCo2;
		},
	},
	watch: {
		offline() {
			this.loadSessions();
		},
	},
	mounted() {
		this.loadSessions();
	},
	methods: {
		changePeriod(newPeriod: PERIODS) {
			let month: number | undefined = this.month;
			let year: number | undefined = this.year;
			let period: PERIODS | undefined = newPeriod;
			switch (period) {
				case PERIODS.TOTAL:
					month = undefined;
					year = undefined;
					break;
				case PERIODS.YEAR:
					month = undefined;
					break;
				default:
					period = undefined;
			}
			this.$router.push({ query: { ...this.$route.query, period, month, year } });
		},
		dateToPeriodKey(date: Date) {
			const options: Intl.DateTimeFormatOptions = {
				year: "numeric",
				month: "numeric",
				day: "numeric",
			};
			if (this.period === PERIODS.YEAR) options.day = undefined;
			if (this.period === PERIODS.TOTAL) options.month = undefined;
			return date.toLocaleDateString(undefined, options);
		},
		async loadSessions() {
			const response = await api.get("sessions");
			// ensure sessions are sorted by created date
			const sortedSessions = response.data?.sort((a: Session, b: Session) => {
				return new Date(a.created).getTime() - new Date(b.created).getTime();
			});
			this.sessions = sortedSessions;
		},
		showDetails(sessionId: number) {
			this.selectedSessionId = sessionId;
			const modal = Modal.getOrCreateInstance(
				document.getElementById("sessionDetailsModal") as HTMLElement
			);
			modal.show();
		},
		csvHrefLink(year?: number, month?: number) {
			const params = new URLSearchParams({
				format: "csv",
				lang: this.$i18n?.locale,
			});
			if (year) params.append("year", year.toString());
			if (month) params.append("month", month.toString());
			return `./api/sessions?${params.toString()}`;
		},
		updateType(type: TYPES) {
			this.selectedType = type;
			settings.sessionsType = type;
		},
		updateGroup(group: GROUPS) {
			this.selectedGroup = group;
			settings.sessionsGroup = group;
		},
		updateDate({ year, month }: { year: number; month: number }) {
			this.$router.push({ query: { ...this.$route.query, year, month } });
		},
		percentile(arr: number[], p: number): number | null {
			if (arr.length === 0) return null;
			const sorted = arr.sort((a, b) => a - b);
			const index = (p / 100) * (sorted.length - 1);
			return sorted[Math.floor(index)] ?? null;
		},
	},
});
</script>

<style scoped>
.header-outer {
	--vertical-shift: 0rem;
	left: 0;
	right: 0;
	top: max(0rem, env(safe-area-inset-top)) !important;
	margin: 0 calc(calc(1.5rem + var(--vertical-shift)) * -1);
	-webkit-backdrop-filter: blur(35px);
	backdrop-filter: blur(35px);
	background-color: #0000;
	box-shadow: 0 1px 8px 0px var(--evcc-background);
}

@media (min-width: 576px) {
	.header-outer {
		--vertical-shift: calc((100vw - 560px) / 2);
	}
}

@media (min-width: 768px) {
	.header-outer {
		--vertical-shift: calc((100vw - 740px) / 2);
	}
}

@media (min-width: 992px) {
	.header-outer {
		--vertical-shift: calc((100vw - 980px) / 2);
	}
}

@media (min-width: 1200px) {
	.header-outer {
		--vertical-shift: calc((100vw - 1160px) / 2);
	}
}

@media (min-width: 1400px) {
	.header-outer {
		--vertical-shift: calc((100vw - 1340px) / 2);
	}
}
</style>
