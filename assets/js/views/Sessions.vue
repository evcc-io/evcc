<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader :title="$t('sessions.title')" />
		<div class="row">
			<main class="col-12">
				<div class="header-outer sticky-top mb-3">
					<div class="container">
						<div class="row py-2 py-sm-3 d-flex flex-column flex-sm-row gap-2 gap-lg-0">
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
				<div class="d-flex justify-content-between align-items-center gap-2">
					<h3
						class="fw-normal my-0 d-flex gap-2 flex-wrap d-flex align-items-baseline overflow-hidden"
					>
						<span class="d-block no-wrap text-truncate">{{ energyTitle }}</span>
						<small class="d-block no-wrap text-truncate">{{ energySubTitle }}</small>
					</h3>
					<IconSelectGroup>
						<IconSelectItem
							v-for="group in Object.values(solarGroups)"
							:key="group"
							:active="selectedSolarGroup === group"
							@click="updateSolarGroup(group)"
						>
							<component :is="solarGroupIcons[group]"></component>
						</IconSelectItem>
					</IconSelectGroup>
				</div>
				<EnergyHistoryChart
					class="mb-5"
					:sessions="currentSessions"
					:color-mappings="colorMappings"
					:group-by="selectedSolarGroup"
					:period="period"
				/>
				<div v-if="showExtraCharts" class="row align-items-start">
					<div class="col-12 col-lg-6 mb-5">
						<h3 class="fw-normal my-4">{{ solarTitle }}</h3>
						<SolarYearChart
							v-if="showSolarYearChart"
							:period="period"
							:sessions="currentSessions"
						/>
						<SolarGroupedChart
							v-else
							:sessions="currentSessions"
							:color-mappings="colorMappings"
							:group-by="selectedSolarGroup"
						/>
					</div>
					<div class="col-12 col-lg-6 mb-5">
						<h3 class="fw-normal my-4">{{ energyGroupedTitle }}</h3>
						<EnergyGroupedChart
							:sessions="currentSessions"
							:color-mappings="colorMappings"
							:group-by="selectedSolarGroup"
						/>
					</div>
				</div>
				<div v-if="showCostCharts">
					<div class="d-flex justify-content-between align-items-center gap-2">
						<h3
							class="fw-normal my-0 d-flex gap-2 flex-wrap d-flex align-items-baseline overflow-hidden"
						>
							<span class="d-block no-wrap text-truncate">{{ costTitle }}</span>
							<small class="d-block no-wrap text-truncate">{{ costSubTitle }}</small>
						</h3>
						<div class="d-flex gap-2">
							<IconSelectGroup v-if="showCostTypeSelector">
								<IconSelectItem
									:active="selectedCostType === costTypes.PRICE"
									@click="updateCostType(costTypes.PRICE)"
								>
									<DynamicPriceIcon />
								</IconSelectItem>
								<IconSelectItem
									:active="selectedCostType === costTypes.CO2"
									@click="updateCostType(costTypes.CO2)"
								>
									<shopicon-regular-eco1></shopicon-regular-eco1>
								</IconSelectItem>
							</IconSelectGroup>
							<IconSelectGroup>
								<IconSelectItem
									v-for="group in Object.values(costGroups)"
									:key="group"
									:active="selectedCostGroup === group"
									@click="updateCostGroup(group)"
								>
									<component :is="costGroupIcons[group]"></component>
								</IconSelectItem>
							</IconSelectGroup>
						</div>
					</div>
					<CostHistoryChart
						class="mb-5"
						:sessions="currentCostTypeSessions"
						:color-mappings="colorMappings"
						:group-by="selectedCostGroup"
						:cost-type="activeCostType"
						:period="period"
					/>
					<div v-if="showExtraCharts" class="row align-items-start">
						<div class="col-12 col-lg-6 mb-5">
							<h3 class="fw-normal my-4">{{ avgCostTitle }}</h3>
							<CostYearChart
								:period="period"
								:sessions="currentCostTypeSessions"
								:cost-type="activeCostType"
							/>
						</div>
						<div class="col-12 col-lg-6 mb-5">
							<h3 class="fw-normal my-4">{{ costGroupedTitle }}</h3>
						</div>
					</div>
				</div>
				<div v-if="showTable">
					<h3>Übersicht</h3>
					<SessionTable
						:sessions="currentSessions"
						:vehicleFilter="vehicleFilter"
						:loadpointFilter="loadpointFilter"
						:currency="currency"
						@show-session="showDetails"
					/>
					<div class="d-grid gap-2 d-md-block mt-3 mb-5">
						<a
							v-if="currentSessions.length"
							class="btn btn-sm btn-outline-secondary text-nowrap me-md-2"
							:href="csvLink"
							download
						>
							{{ $t("sessions.csvMonth", { month: headline }) }}
						</a>
						<a
							v-if="sessions.length"
							class="btn btn-sm btn-outline-secondary text-nowrap"
							:href="csvTotalLink"
							download
						>
							{{ $t("sessions.csvTotal") }}
						</a>
					</div>
				</div>
			</main>
			<SessionDetailsModal
				:session="selectedSession"
				:vehicles="vehicleList"
				:currency="currency"
				@session-changed="loadSessions"
			/>
		</div>
	</div>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";
import "@h2d2/shopicons/es/regular/cablecharge";
import "@h2d2/shopicons/es/regular/car3";
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
import CostYearChart from "../components/Sessions/CostYearChart.vue";
import TopHeader from "../components/TopHeader.vue";
import IconSelectGroup from "../components/IconSelectGroup.vue";
import IconSelectItem from "../components/IconSelectItem.vue";
import SelectGroup from "../components/SelectGroup.vue";
import CustomSelect from "../components/CustomSelect.vue";
import colors from "../colors";
import settings from "../settings";
import PeriodSelector from "../components/Sessions/PeriodSelector.vue";
import DateNavigator from "../components/Sessions/DateNavigator.vue";
import DynamicPriceIcon from "../components/MaterialIcon/DynamicPrice.vue";
import TotalIcon from "../components/MaterialIcon/Total.vue";

const COST_TYPES = {
	PRICE: "price",
	CO2: "co2",
};

const SOLAR_GROUPS = {
	NONE: "none",
	LOADPOINT: "loadpoint",
	VEHICLE: "vehicle",
};

const COST_GROUPS = {
	NONE: "none",
	LOADPOINT: "loadpoint",
	VEHICLE: "vehicle",
};

const PERIODS = {
	MONTH: "month",
	YEAR: "year",
	TOTAL: "total",
};

export default {
	name: "Sessions",
	components: {
		SessionDetailsModal,
		SessionTable,
		TopHeader,
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
		CostYearChart,
	},
	mixins: [formatter],
	props: {
		notifications: Array,
		month: { type: Number, default: () => new Date().getMonth() + 1 },
		year: { type: Number, default: () => new Date().getFullYear() },
		period: { type: String, default: PERIODS.MONTH },
		loadpointFilter: { type: String, default: "" },
		vehicleFilter: { type: String, default: "" },
		offline: Boolean,
	},
	data() {
		return {
			sessions: [],
			selectedSolarGroup: settings.sessionsSolarGroup || SOLAR_GROUPS.NONE,
			selectedCostGroup: settings.sessionsCostGroup || COST_GROUPS.NONE,
			selectedCostType: settings.sessionsCostType || COST_TYPES.PRICE,
			selectedSessionId: undefined,
			periods: PERIODS,
			solarGroups: SOLAR_GROUPS,
			costGroups: COST_GROUPS,
			costTypes: COST_TYPES,
		};
	},
	head() {
		return { title: `${this.$t("sessions.title")} | evcc` };
	},
	computed: {
		currency() {
			return store.state.currency || "EUR";
		},
		energyTitle() {
			if (this.selectedSolarGroup === SOLAR_GROUPS.VEHICLE) {
				return "Fahrzeuge";
			} else if (this.selectedSolarGroup === SOLAR_GROUPS.LOADPOINT) {
				return "Ladepunkte";
			} else {
				return `${this.solarPercentageFmt} Sonne`;
			}
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
			return `${this.energySumFmt} gesamt`;
		},
		solarTitle() {
			return `${this.solarPercentageFmt} Sonnenanteil`;
		},
		energyGroupedTitle() {
			return `${this.energySumFmt} Energiemenge`;
		},
		avgCostTitle() {
			return "Ladepreis";
		},
		costGroupedTitle() {
			return "Ladepunkt";
		},
		periodOptions() {
			return Object.entries(PERIODS).map(([key, value]) => ({
				name: this.$t(`sessions.period.${key.toLowerCase()}`),
				value,
			}));
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
		currentCostTypeSessions() {
			if (this.activeCostType === COST_TYPES.PRICE) {
				return this.currentSessionsWithPrice;
			} else {
				return this.currentSessionsWithCo2;
			}
		},
		totalPrice() {
			return this.currentSessionsWithPrice.reduce((acc, s) => acc + s.price, 0);
		},
		pricePerKWh() {
			const list = this.currentSessionsWithPrice;
			const energy = list.reduce((acc, s) => acc + s.chargedEnergy, 0);
			return energy ? this.totalPrice / energy : null;
		},
		currentSessionsWithCo2() {
			return this.currentSessions.filter((s) => s.co2PerKWh !== null);
		},
		totalCo2() {
			const list = this.currentSessionsWithCo2;
			return list.reduce((acc, s) => acc + s.co2PerKWh * s.chargedEnergy, 0);
		},
		co2PerKWh() {
			const list = this.currentSessionsWithCo2;
			const energy = list.reduce((acc, s) => acc + s.chargedEnergy, 0);
			return energy ? this.totalCo2 / energy : null;
		},
		costTitle() {
			return this.activeCostType === COST_TYPES.PRICE
				? `${this.fmtPricePerKWh(this.pricePerKWh, this.currency)} Ladepreis`
				: `${this.fmtCo2Medium(this.co2PerKWh)} CO₂-Emission`;
		},
		costSubTitle() {
			return this.activeCostType === COST_TYPES.PRICE
				? `${this.fmtMoney(this.totalPrice, this.currency, true, true)} gesamt`
				: `${this.fmtGrams(this.totalCo2)} gesamt`;
		},
		activeCostType() {
			if (this.selectedCostType === COST_TYPES.PRICE && this.costTypePriceAvailable) {
				return COST_TYPES.PRICE;
			} else if (this.selectedCostType === COST_TYPES.CO2 && this.costTypeCo2Available) {
				return COST_TYPES.CO2;
			}
			return null;
		},
		showCostCharts() {
			return this.costTypePriceAvailable || this.costTypeCo2Available;
		},
		costTypePriceAvailable() {
			return this.currentSessionsWithPrice.length > 0;
		},
		costTypeCo2Available() {
			return this.currentSessionsWithCo2.length > 0;
		},
		showCostTypeSelector() {
			return this.costTypePriceAvailable && this.costTypeCo2Available;
		},
		startDate() {
			return new Date(this.sessions[0]?.created || Date.now());
		},
		topNavigation: function () {
			const vehicleLogins = store.state.auth ? store.state.auth.vehicles : {};
			return { vehicleLogins, ...this.collectProps(TopNavigation, store.state) };
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
			return Object.entries(vehicles).map(([name, vehicle]) => ({ name, ...vehicle }));
		},
		selectedSession() {
			return this.sessions.find((s) => s.id == this.selectedSessionId);
		},
		monthName() {
			const date = new Date();
			date.setMonth(this.month - 1, 1);
			return this.fmtMonth(date);
		},
		headline() {
			const date = new Date();
			date.setMonth(this.month - 1, 1);
			date.setFullYear(this.year);
			return this.fmtMonthYear(date);
		},
		csvLink() {
			return this.csvHrefLink(this.year, this.month);
		},
		csvTotalLink() {
			return this.csvHrefLink();
		},
		colorMappings() {
			const lastThreeMonths = new Date();
			lastThreeMonths.setMonth(lastThreeMonths.getMonth() - 3);

			// Aggregate energy to get sorted list of loadpoints/vehicles for coloring
			const aggregateEnergy = (group) => {
				return this.sessionsWithDefaults.reduce((acc, session) => {
					if (new Date(session.created) >= lastThreeMonths) {
						const key = session[group];
						acc[key] = (acc[key] || 0) + session.chargedEnergy;
					}
					return acc;
				}, {});
			};

			// Assign colors based on energy usage
			const assignColors = (energyAggregation, colorType) => {
				const result = {};
				let colorIndex = 0;

				// Assign colors by used energy in the last three months
				const sortedEntries = Object.entries(energyAggregation).sort((a, b) => b[1] - a[1]);
				sortedEntries.forEach(([key]) => {
					if (!result[key]) {
						result[key] = colors.palette[colorIndex % colors.palette.length];
						colorIndex++;
					}
				});

				// Assign colors to remaining entries
				this.sessionsWithDefaults.forEach((session) => {
					const key = session[colorType];
					if (!result[key]) {
						result[key] = colors.palette[colorIndex % colors.palette.length];
						colorIndex++;
					}
				});

				return result;
			};

			const loadpointEnergy = aggregateEnergy("loadpoint");
			const loadpointColors = assignColors(loadpointEnergy, "loadpoint");

			const vehicleEnergy = aggregateEnergy("vehicle");
			const vehicleColors = assignColors(vehicleEnergy, "vehicle");

			const solar = { self: colors.self, grid: colors.grid };
			const cost = { price: colors.price, co2: colors.co2 };

			return {
				loadpoint: loadpointColors,
				vehicle: vehicleColors,
				solar,
				cost,
			};
		},
		solarGroupIcons() {
			return {
				[SOLAR_GROUPS.NONE]: "shopicon-regular-sun",
				[SOLAR_GROUPS.LOADPOINT]: "shopicon-regular-cablecharge",
				[SOLAR_GROUPS.VEHICLE]: "shopicon-regular-car3",
			};
		},
		costTypeIcons() {
			return {
				[COST_TYPES.PRICE]: DynamicPriceIcon,
				[COST_TYPES.CO2]: "shopicon-regular-eco1",
			};
		},
		costGroupIcons() {
			return {
				[COST_GROUPS.NONE]: this.showCostTypeSelector
					? TotalIcon
					: this.costTypePriceAvailable
						? DynamicPriceIcon
						: "shopicon-regular-eco1",
				[COST_GROUPS.LOADPOINT]: "shopicon-regular-cablecharge",
				[COST_GROUPS.VEHICLE]: "shopicon-regular-car3",
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
		yearOptions() {
			if (this.sessions.length === 0) {
				return [];
			}
			const first = new Date(this.sessions[0].created);
			const last = new Date();
			const years = [];
			for (let year = first.getFullYear(); year <= last.getFullYear(); year++) {
				years.push({ name: year, value: year });
			}
			return years;
		},
		monthOptions() {
			return Array.from({ length: 12 }, (_, i) => i + 1).map((month) => ({
				name: this.fmtMonth(new Date(this.year, month - 1, 1)),
				value: month,
			}));
		},
		monthYearOptions() {
			if (this.sessions.length === 0) {
				return [];
			}
			const first = new Date(this.sessions[0].created);
			const last = new Date();
			const yearMonths = [];
			for (let year = first.getFullYear(); year <= last.getFullYear(); year++) {
				const startMonth = year === first.getFullYear() ? first.getMonth() + 1 : 1;
				const endMonth = year === last.getFullYear() ? last.getMonth() + 1 : 12;
				for (let month = startMonth; month <= endMonth; month++) {
					yearMonths.push({
						name: this.fmtMonthYear(new Date(year, month - 1, 1)),
						value: `${year}-${month}`,
					});
				}
			}
			return yearMonths;
		},
		groupEntriesAvailable() {
			if (this.selectedSolarGroup === SOLAR_GROUPS.NONE || !this.currentSessions.length)
				return false;
			return new Set(this.currentSessions.map((s) => s[this.selectedSolarGroup])).size > 1;
		},
		showSolarYearChart() {
			return (
				this.showExtraCharts &&
				this.period !== PERIODS.MONTH &&
				this.selectedSolarGroup === SOLAR_GROUPS.NONE
			);
		},
		showExtraCharts() {
			if (this.period === PERIODS.MONTH && this.selectedSolarGroup === SOLAR_GROUPS.NONE) {
				return false;
			}
			if (
				[SOLAR_GROUPS.LOADPOINT, SOLAR_GROUPS.VEHICLE].includes(this.selectedSolarGroup) &&
				!this.groupEntriesAvailable
			) {
				return false;
			}
			return true;
		},
	},
	watch: {
		offline: function () {
			this.loadSessions();
		},
	},
	mounted() {
		this.loadSessions();
	},
	methods: {
		changePeriod(newPeriod) {
			let month = this.month;
			let year = this.year;
			let period = newPeriod;
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
		async loadSessions() {
			const response = await api.get("sessions");
			// ensure sessions are sorted by created date
			const sortedSessions = response.data?.result.sort((a, b) => {
				return new Date(a.created) - new Date(b.created);
			});
			this.sessions = sortedSessions;
		},
		showDetails(sessionId) {
			this.selectedSessionId = sessionId;
			const modal = Modal.getOrCreateInstance(document.getElementById("sessionDetailsModal"));
			modal.show();
		},
		csvHrefLink(year, month) {
			const params = new URLSearchParams({
				format: "csv",
				lang: this.$i18n?.locale,
			});
			if (year && month) {
				params.append("year", year);
				params.append("month", month);
			}
			return `./api/sessions?${params.toString()}`;
		},
		updateSolarGroup(group) {
			this.selectedSolarGroup = group;
			settings.sessionsSolarGroup = group;
		},
		updateCostType(costType) {
			this.selectedCostType = costType;
			settings.sessionsCostType = costType;
		},
		updateCostGroup(group) {
			this.selectedCostGroup = group;
			settings.sessionsCostGroup = group;
		},
		updateDate({ year, month }) {
			this.$router.push({ query: { ...this.$route.query, year, month } });
		},
	},
};
</script>

<style scoped>
.header-outer {
	--vertical-shift: 0rem;
	left: 0;
	right: 0;
	margin: 0 calc(calc(1.5rem + var(--vertical-shift)) * -1);
	-webkit-backdrop-filter: blur(35px);
	backdrop-filter: blur(35px);
	background-color: #0000;
	box-shadow: 0 1px 8px 0px var(--evcc-background);
}

@media (min-width: 576px) {
	.header-outer {
		--vertical-shift: calc((100vw - 540px) / 2);
	}
}

@media (min-width: 768px) {
	.header-outer {
		--vertical-shift: calc((100vw - 720px) / 2);
	}
}

@media (min-width: 992px) {
	.header-outer {
		--vertical-shift: calc((100vw - 960px) / 2);
	}
}

@media (min-width: 1200px) {
	.header-outer {
		--vertical-shift: calc((100vw - 1140px) / 2);
	}
}

@media (min-width: 1400px) {
	.header-outer {
		--vertical-shift: calc((100vw - 1320px) / 2);
	}
}
</style>
