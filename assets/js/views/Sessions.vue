<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader :title="$t('sessions.title')" />
		<div class="row">
			<main class="col-12">
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
							:active="selectedGroup === 'loadpoint'"
							@click="selectedGroup = 'loadpoint'"
						>
							<shopicon-regular-cablecharge></shopicon-regular-cablecharge>
						</IconSelectItem>
						<IconSelectItem
							:active="selectedGroup === 'vehicle'"
							@click="selectedGroup = 'vehicle'"
						>
							<shopicon-regular-car3></shopicon-regular-car3>
						</IconSelectItem>
					</IconSelectGroup>
				</div>
				<EnergyHistoryChart
					class="mb-5"
					:sessions="currentSessions"
					:color-mappings="colorMappings"
					:group-by="selectedGroup"
				/>
				<div class="row align-items-start">
					<EnergyAggregateEntries
						class="col-12 col-md-8 mb-5"
						:sessions="currentSessions"
						:color-mappings="colorMappings"
						:group-by="selectedGroup"
					/>
					<EnergyAggregateChart
						class="col-12 col-md-4 mb-5"
						:sessions="currentSessions"
						:color-mappings="colorMappings"
						:group-by="selectedGroup"
					/>
				</div>
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
import formatter from "../mixins/formatter";
import api from "../api";
import store from "../store";
import SessionDetailsModal from "../components/Sessions/SessionDetailsModal.vue";
import SessionTable from "../components/Sessions/SessionTable.vue";
import EnergyHistoryChart from "../components/Sessions/EnergyHistoryChart.vue";
import EnergyAggregateChart from "../components/Sessions/EnergyAggregateChart.vue";
import EnergyAggregateEntries from "../components/Sessions/EnergyAggregateEntries.vue";
import TopHeader from "../components/TopHeader.vue";
import IconSelectGroup from "../components/IconSelectGroup.vue";
import IconSelectItem from "../components/IconSelectItem.vue";

// const COLORS = [ "#40916C", "#52B788", "#74C69D", "#95D5B2", "#B7E4C7", "#D8F3DC", "#081C15", "#1B4332", "#2D6A4F"];
// const COLORS = ["#577590", "#43AA8B", "#90BE6D", "#F9C74F", "#F8961E", "#F3722C", "#F94144"];
// const COLORS = ["#0077b6", "#00b4d8", "#90e0ef", "#caf0f8", "#03045e"];
// const COLORS = [ "#0077B6FF", "#0096C7FF", "#00B4D8FF", "#48CAE4FF", "#90E0EFFF", "#ADE8F4FF", "#CAF0F8FF", "#03045EFF", "#023E8AFF",

const COLORS = [
	"#0077B6FF",
	"#00B4D8FF",
	"#90E0EFFF",
	"#006769FF",
	"#40A578FF",
	"#9DDE8BFF",
	"#F8961EFF",
	"#F9C74FFF",
	"#E6FF94FF",
];

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
	},
	mixins: [formatter],
	props: {
		notifications: Array,
		month: { type: Number, default: () => new Date().getMonth() + 1 },
		year: { type: Number, default: () => new Date().getFullYear() },
		loadpointFilter: { type: String, default: "" },
		vehicleFilter: { type: String, default: "" },
	},
	data() {
		return {
			sessions: [],
			selectedGroup: "loadpoint",
			selectedSessionId: undefined,
		};
	},
	head() {
		return { title: `${this.$t("sessions.title")} | evcc` };
	},
	computed: {
		energyTitle() {
			const energy =
				this.currentSessions.reduce((acc, session) => acc + session.chargedEnergy, 0) * 1e3;
			return `Geladene Energie: ${this.fmtKWh(energy)}`;
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
				return date.getFullYear() === this.year && date.getMonth() + 1 === this.month;
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
			const first = new Date(this.sessions[length - 1].created);
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
						result[key] = COLORS[colorIndex % COLORS.length];
						colorIndex++;
					}
				});

				// Assign colors to remaining entries
				this.sessionsWithDefaults.forEach((session) => {
					const key = session[colorType];
					if (!result[key]) {
						result[key] = COLORS[colorIndex % COLORS.length];
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
	},
	mounted() {
		this.loadSessions();
	},
	methods: {
		async loadSessions() {
			const response = await api.get("sessions");
			this.sessions = response.data?.result;
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
	},
};
</script>

<style scoped>
.month-header {
	background-color: var(--evcc-background);
}
</style>
