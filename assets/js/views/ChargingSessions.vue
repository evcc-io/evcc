<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader :title="$t('sessions.title')" />
		<div class="row">
			<main class="col-12">
				<div
					class="d-flex align-items-baseline justify-content-between my-3 my-md-5 month-header"
				>
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
					<h2 class="text-center">{{ headline }}</h2>
					<router-link
						class="d-flex text-decoration-none align-items-center"
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
				<ChargingSessionTable
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
			<ChargingSessionModal
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
import formatter from "../mixins/formatter";
import api from "../api";
import store from "../store";
import ChargingSessionModal from "../components/ChargingSessionModal.vue";
import ChargingSessionTable from "../components/ChargingSessionTable.vue";
import settings from "../settings";
import TopHeader from "../components/TopHeader.vue";

export default {
	name: "ChargingSessions",
	components: { ChargingSessionModal, ChargingSessionTable, TopHeader },
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
			selectedSessionId: undefined,
		};
	},
	head() {
		return { title: `${this.$t("sessions.title")} | evcc` };
	},
	computed: {
		topNavigation: function () {
			const vehicleLogins = store.state.auth ? store.state.auth.vehicles : {};
			return { vehicleLogins, ...this.collectProps(TopNavigation, store.state) };
		},
		currentSessions() {
			const sessionsWithDefaults = this.sessions.map((session) => {
				const loadpoint = session.loadpoint || this.$t("main.loadpoint.fallbackName");
				const vehicle = session.vehicle || this.$t("main.vehicle.unknown");
				return { ...session, loadpoint, vehicle };
			});

			return sessionsWithDefaults.filter((session) => {
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
