<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader :title="$t('sessions.title')" />
		<div class="row">
			<main class="col-12">
				<div class="row mt-2 mb-3">
					<div
						class="col-lg-5 d-flex"
						:class="showMonthNavigation || showYearNavigation ? 'mb-3' : 'mb-4'"
					>
						<SelectGroup
							id="sessionsPeriod"
							class="w-100"
							:options="[
								{ name: 'Monat', value: 'month' },
								{ name: 'Jahr', value: 'year' },
								{ name: 'Gesamt', value: 'total' },
							]"
							:modelValue="period"
							@update:modelValue="changePeriod"
						/>
					</div>
					<div
						v-if="showMonthNavigation || showYearNavigation"
						class="col-lg-6 mb-3 offset-lg-1 d-sm-flex justify-content-lg-end gap-lg-4"
						:class="
							showMonthNavigation ? 'justify-content-between' : 'justify-content-end'
						"
					>
						<div
							v-if="showMonthNavigation"
							class="d-none d-sm-flex mb-2 mb-lg-0 justify-content-between"
						>
							<button
								class="btn btn-sm border-0"
								:disabled="!hasPrevMonth"
								@click="navigatePrevMonth"
							>
								<shopicon-regular-angledoubleleftsmall
									size="s"
									class="me-1"
								></shopicon-regular-angledoubleleftsmall>
							</button>
							<CustomSelect
								id="sessionsMonth"
								:options="monthOptions"
								:selected="month"
								@change="navigateMonth($event.target.value)"
							>
								<button
									class="btn btn-sm border-0 text-truncate h-100"
									style="width: 8em"
								>
									{{ monthName }}
								</button>
							</CustomSelect>
							<button
								class="btn btn-sm border-0"
								:disabled="!hasNextMonth"
								@click="navigateNextMonth"
							>
								<shopicon-regular-angledoublerightsmall
									size="s"
									class="ms-1"
								></shopicon-regular-angledoublerightsmall>
							</button>
						</div>
						<div
							v-if="showMonthNavigation"
							class="d-flex d-sm-none mb-2 mb-lg-0 justify-content-between"
						>
							<button
								class="btn btn-sm border-0"
								:disabled="!hasPrevMonth"
								@click="navigatePrevMonth"
							>
								<shopicon-regular-angledoubleleftsmall
									size="s"
									class="me-1"
								></shopicon-regular-angledoubleleftsmall>
							</button>
							<CustomSelect
								id="sessionsMonthYear"
								:options="monthYearOptions"
								:selected="month"
								@change="navigateMonthYear($event.target.value)"
							>
								<button class="btn btn-sm border-0 h-100 text-truncate">
									{{ headline }}
								</button>
							</CustomSelect>
							<button
								class="btn btn-sm border-0"
								:disabled="!hasNextMonth"
								@click="navigateNextMonth"
							>
								<shopicon-regular-angledoublerightsmall
									size="s"
									class="ms-1"
								></shopicon-regular-angledoublerightsmall>
							</button>
						</div>
						<div
							v-if="showYearNavigation"
							class="mb-2 mb-lg-0 justify-content-between"
							:class="showMonthNavigation ? 'd-none d-sm-flex' : 'd-flex'"
						>
							<button
								class="btn btn-sm border-0"
								:disabled="!hasPrevYear"
								@click="navigatePrevYear"
							>
								<shopicon-regular-angledoubleleftsmall
									size="s"
									class="me-1"
								></shopicon-regular-angledoubleleftsmall>
							</button>
							<CustomSelect
								id="sessionsYear"
								:options="yearOptions"
								:selected="year"
								@change="navigateYear($event.target.value)"
							>
								<button class="btn btn-sm border-0 h-100" style="width: 4em">
									{{ year }}
								</button>
							</CustomSelect>
							<button
								class="btn btn-sm border-0"
								:disabled="!hasNextYear"
								@click="navigateNextYear"
							>
								<shopicon-regular-angledoublerightsmall
									size="s"
									class="ms-1"
								></shopicon-regular-angledoublerightsmall>
							</button>
						</div>
					</div>
				</div>
				<div class="d-flex justify-content-between align-items-center">
					<h3 class="fw-normal my-0 d-flex gap-2 flex-wrap">
						<span class="d-block no-wrap">{{ energyTitle }}</span>
						<small class="d-block no-wrap">{{ energySubTitle }}</small>
					</h3>
					<IconSelectGroup>
						<IconSelectItem
							v-for="group in Object.values(groups)"
							:key="group"
							:active="selectedGroup === group"
							@click="updateGroup(group)"
						>
							<component :is="groupIcons[group]"></component>
						</IconSelectItem>
					</IconSelectGroup>
				</div>
				<EnergyHistoryChart
					class="mb-5"
					:sessions="currentSessions"
					:color-mappings="colorMappings"
					:group-by="selectedGroup"
					:period="period"
				/>
				<div class="row align-items-start" v-if="selectedGroup !== groups.SOLAR">
					<div class="col-12 col-md-6 mb-5">
						<h3 class="fw-normal my-4">Sonnenanteil</h3>
						<SolarChart
							:sessions="currentSessions"
							:color-mappings="colorMappings"
							:group-by="selectedGroup"
						/>
					</div>
					<div class="col-12 col-md-6 mb-5">
						<h3 class="fw-normal my-4">Energiemenge</h3>
						<EnergyAggregateChart
							:sessions="currentSessions"
							:color-mappings="colorMappings"
							:group-by="selectedGroup"
						/>
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
import "@h2d2/shopicons/es/regular/angledoubleleftsmall";
import "@h2d2/shopicons/es/regular/angledoublerightsmall";
import "@h2d2/shopicons/es/regular/cablecharge";
import "@h2d2/shopicons/es/regular/car3";
import "@h2d2/shopicons/es/regular/sun";
import formatter, { POWER_UNIT } from "../mixins/formatter";
import api from "../api";
import store from "../store";
import SessionDetailsModal from "../components/Sessions/SessionDetailsModal.vue";
import SessionTable from "../components/Sessions/SessionTable.vue";
import EnergyHistoryChart from "../components/Sessions/EnergyHistoryChart.vue";
import EnergyAggregateChart from "../components/Sessions/EnergyAggregateChart.vue";
import EnergyAggregateEntries from "../components/Sessions/EnergyAggregateEntries.vue";
import SolarChart from "../components/Sessions/SolarChart.vue";
import TopHeader from "../components/TopHeader.vue";
import IconSelectGroup from "../components/IconSelectGroup.vue";
import IconSelectItem from "../components/IconSelectItem.vue";
import SelectGroup from "../components/SelectGroup.vue";
import CustomSelect from "../components/CustomSelect.vue";
import colors from "../colors";
import settings from "../settings";

const GROUPS = {
	SOLAR: "solar",
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
		EnergyAggregateChart,
		EnergyAggregateEntries,
		IconSelectGroup,
		IconSelectItem,
		SelectGroup,
		CustomSelect,
		SolarChart,
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
			selectedGroup: settings.sessionsGroup || GROUPS.SOLAR,
			selectedSessionId: undefined,
			groups: GROUPS,
		};
	},
	head() {
		return { title: `${this.$t("sessions.title")} | evcc` };
	},
	computed: {
		energyTitle() {
			const totalEnergy = this.totalEnergy;
			const selfEnergy = this.selfEnergy;
			const solarPercentage = totalEnergy > 0 ? (100 / totalEnergy) * selfEnergy : 0;
			return `${this.fmtPercentage(solarPercentage)} Sonne`;
		},
		energySubTitle() {
			return `${this.fmtWh(this.totalEnergy * 1e3, POWER_UNIT.AUTO)} gesamt`;
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
		currency() {
			return store.state.currency;
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
		prevMonthDate() {
			const date = new Date();
			date.setFullYear(this.year);
			date.setMonth(this.month - 2, 1);
			return date;
		},
		prevYearMonth() {
			return {
				year: this.prevMonthDate.getFullYear(),
				month: this.prevMonthDate.getMonth() + 1,
			};
		},
		nextMonthDate() {
			const date = new Date();
			date.setFullYear(this.year);
			date.setMonth(this.month, 1);
			return date;
		},
		nextYearMonth() {
			return {
				year: this.nextMonthDate.getFullYear(),
				month: this.nextMonthDate.getMonth() + 1,
			};
		},
		nextYear() {
			const date = new Date();
			date.setFullYear(this.year + 1);
			return date.getFullYear();
		},
		prevYear() {
			const date = new Date();
			date.setFullYear(this.year - 1);
			return date.getFullYear();
		},
		hasNextMonth() {
			const now = new Date();
			return this.year < now.getFullYear() || this.month < now.getMonth() + 1;
		},
		hasPrevMonth() {
			if (this.sessions.length === 0) {
				return false;
			}
			const first = new Date(this.sessions[0].created);
			return this.year > first.getFullYear() || this.month > first.getMonth() + 1;
		},
		hasNextYear() {
			return this.year < new Date().getFullYear();
		},
		hasPrevYear() {
			if (this.sessions.length === 0) {
				return false;
			}
			const last = new Date(this.sessions[0].created);
			return this.year > last.getFullYear();
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

			return { loadpoint: loadpointColors, vehicle: vehicleColors };
		},
		groupIcons() {
			return {
				[GROUPS.SOLAR]: "shopicon-regular-sun",
				[GROUPS.LOADPOINT]: "shopicon-regular-cablecharge",
				[GROUPS.VEHICLE]: "shopicon-regular-car3",
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
		updateGroup(group) {
			this.selectedGroup = group;
			settings.sessionsGroup = group;
		},
		navigateNextYear() {
			this.$router.push({ query: { ...this.$route.query, year: this.nextYear } });
		},
		navigatePrevYear() {
			this.$router.push({ query: { ...this.$route.query, year: this.prevYear } });
		},
		navigateNextMonth() {
			this.$router.push({ query: { ...this.$route.query, ...this.nextYearMonth } });
		},
		navigatePrevMonth() {
			this.$router.push({ query: { ...this.$route.query, ...this.prevYearMonth } });
		},
		navigateYear(year) {
			this.$router.push({ query: { ...this.$route.query, year } });
		},
		navigateMonth(month) {
			this.$router.push({ query: { ...this.$route.query, month } });
		},
		navigateMonthYear(monthYear) {
			const [year, month] = monthYear.split("-");
			this.$router.push({ query: { ...this.$route.query, year, month } });
		},
	},
};
</script>

<style scoped>
.month-header {
	background-color: var(--evcc-background);
}
</style>
