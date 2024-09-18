<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader :title="$t('sessions.title')" />
		<div class="row">
			<main class="col-12">
				<div class="d-flex justify-content-center">
					<SelectGroup
						id="period"
						:options="[
							{ name: 'Monat', value: 'month' },
							{ name: 'Jahr', value: 'year' },
							{ name: 'Gesamt', value: 'total' },
						]"
						v-model="selectedPeriod"
					/>
				</div>
				<div class="row mt-2 mt-md-3 mb-5 month-header pt-3 pb-2 sticky-top">
					<div class="col-4 d-flex justify-content-start align-items-center">
						<router-link
							class="d-flex text-decoration-none align-items-center"
							:class="{ 'pe-none': !hasPrev, 'text-muted': !hasPrev }"
							:to="{ query: { ...$route.query, ...prevYearMonth } }"
						>
							<shopicon-regular-angledoubleleftsmall
								size="s"
								class="me-1"
							></shopicon-regular-angledoubleleftsmall>
							<span class="d-none d-sm-block">{{ prevMonthName }}</span>
							<span class="d-block d-sm-none">{{ prevMonthNameShort }}</span>
						</router-link>
					</div>
					<h2 class="col-4 text-center">{{ headline }}</h2>
					<div class="col-4 d-flex justify-content-end align-items-center">
						<router-link
							class="d-flex text-decoration-none"
							:class="{ 'pe-none': !hasNext, 'text-muted': !hasNext }"
							:to="{ query: { ...$route.query, ...nextYearMonth } }"
						>
							<span class="d-none d-sm-block">{{ nextMonthName }}</span>
							<span class="d-block d-sm-none">{{ nextMonthNameShort }}</span>
							<shopicon-regular-angledoublerightsmall
								size="s"
								class="ms-1"
							></shopicon-regular-angledoublerightsmall>
						</router-link>
					</div>
				</div>
				<div class="d-flex justify-content-between align-items-center">
					<h3 class="fw-normal my-0">{{ energyTitle }}</h3>
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
					:period="selectedPeriod"
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
				<div v-if="selectedPeriod === 'month'">
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
							class="btn btn-outline-secondary text-nowrap me-md-2"
							:href="csvLink"
							download
						>
							{{ $t("sessions.csvMonth", { month: headline }) }}
						</a>
						<a
							v-if="sessions.length"
							class="btn btn-outline-secondary text-nowrap"
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
import formatter from "../mixins/formatter";
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
import colors from "../colors";
import settings from "../settings";

const GROUPS = {
	SOLAR: "solar",
	LOADPOINT: "loadpoint",
	VEHICLE: "vehicle",
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
		SolarChart,
	},
	mixins: [formatter],
	props: {
		notifications: Array,
		month: { type: Number, default: () => new Date().getMonth() + 1 },
		year: { type: Number, default: () => new Date().getFullYear() },
		loadpointFilter: { type: String, default: "" },
		vehicleFilter: { type: String, default: "" },
		offline: Boolean,
	},
	data() {
		return {
			sessions: [],
			selectedGroup: settings.sessionsGroup || GROUPS.SOLAR,
			selectedSessionId: undefined,
			selectedPeriod: "month",
			groups: GROUPS,
		};
	},
	head() {
		return { title: `${this.$t("sessions.title")} | evcc` };
	},
	computed: {
		energyTitle() {
			if (this.currentSessions.length === 0) {
				return "";
			}
			const totalEnergy = this.currentSessions.reduce(
				(acc, session) => acc + session.chargedEnergy,
				0
			);
			const selfEnergy = this.currentSessions.reduce(
				(acc, session) => acc + (session.chargedEnergy / 100) * session.solarPercentage,
				0
			);
			const solarPercentage = (100 / totalEnergy) * selfEnergy;
			console.log({ totalEnergy, selfEnergy, solarPercentage });
			return `${this.fmtKWh(totalEnergy * 1e3)} geladen · ${this.fmtPercentage(solarPercentage)} Sonne`;
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
				switch (this.selectedPeriod) {
					case "month":
						return (
							date.getFullYear() === this.year && date.getMonth() + 1 === this.month
						);
					case "year":
						return date.getFullYear() === this.year;
					case "total":
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
		prevDate() {
			const date = new Date();
			date.setFullYear(this.year);
			date.setMonth(this.month - 2, 1);
			return date;
		},
		prevYearMonth() {
			return { year: this.prevDate.getFullYear(), month: this.prevDate.getMonth() + 1 };
		},
		prevMonthName() {
			return this.fmtMonth(this.prevDate);
		},
		prevMonthNameShort() {
			return this.fmtMonth(this.prevDate, true);
		},
		nextDate() {
			const date = new Date();
			date.setFullYear(this.year);
			date.setMonth(this.month, 1);
			return date;
		},
		nextYearMonth() {
			return { year: this.nextDate.getFullYear(), month: this.nextDate.getMonth() + 1 };
		},
		nextMonthName() {
			return this.fmtMonth(this.nextDate);
		},
		nextMonthNameShort() {
			return this.fmtMonth(this.nextDate, true);
		},
		hasNext() {
			const now = new Date();
			return this.year < now.getFullYear() || this.month < now.getMonth() + 1;
		},
		hasPrev() {
			const length = this.sessions.length;
			if (length === 0) {
				return false;
			}
			const first = new Date(this.sessions[0].created);
			return this.year > first.getFullYear() || this.month > first.getMonth() + 1;
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
	},
};
</script>

<style scoped>
.month-header {
	background-color: var(--evcc-background);
}
</style>
