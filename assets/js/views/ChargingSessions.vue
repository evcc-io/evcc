<template>
	<div class="container px-4">
		<header class="d-flex justify-content-between align-items-center py-3">
			<h1 class="mb-1 pt-1 d-flex text-nowrap">
				<router-link class="dropdown-item mx-2 me-2" to="/">
					<shopicon-bold-arrowback size="s" class="back"></shopicon-bold-arrowback>
				</router-link>
				{{ $t("sessions.title") }}
			</h1>
			<TopNavigation />
		</header>

		<div class="row">
			<main class="col-12">
				<div class="d-flex align-items-baseline justify-content-between my-5">
					<router-link class="d-flex text-decoration-none align-items-center" to="/">
						<shopicon-regular-angledoubleleftsmall
							size="s"
							class="me-1"
						></shopicon-regular-angledoubleleftsmall>
						April
					</router-link>
					<h2 class="text-center">Mai 2023</h2>
					<router-link class="d-flex text-decoration-none align-items-center" to="/">
						Juni
						<shopicon-regular-angledoublerightsmall
							size="s"
							class="ms-1"
						></shopicon-regular-angledoublerightsmall>
					</router-link>
				</div>

				<!--
				<div v-for="group in sessionsByLoadpoint" :key="group.month">
					<div class="d-flex align-items-center my-5">
						<h2 class="me-4 mb-0">
							{{ formatGroupHeadline(group.month) }}
						</h2>
						<a
							class="btn btn-xs btn-outline-secondary text-nowrap"
							:href="csvHrefLink(group.month)"
							download="sessions.csv"
						>
							CSV
						</a>
					</div>
					-->

				<div class="table-responsive my-3">
					<table class="table text-nowrap">
						<thead>
							<tr>
								<th scope="col" class="align-top ps-0">
									{{ $t("sessions.date") }}
								</th>
								<th scope="col" class="align-top">
									{{ $t("sessions.loadpoint") }}
									<label class="position-relative d-block">
										<select
											:value="loadpointFilter"
											class="custom-select"
											@change="changeLoadpointFilter"
										>
											<option
												v-for="{ name, value } in loadpointFilterOptions"
												:key="value"
												:value="value"
											>
												{{ name }}
											</option>
										</select>
										<span
											class="fw-normal text-decoration-underline text-nowrap text-muted pe-none"
										>
											{{ loadpointFilter || $t("sessions.filter.filter") }}
										</span>
									</label>
								</th>
								<th scope="col" class="align-top">
									{{ $t("sessions.vehicle") }}
									<label class="position-relative d-block">
										<select
											:value="vehicleFilter"
											class="custom-select"
											@change="changeVehicleFilter"
										>
											<option
												v-for="{ name, value } in vehicleFilterOptions"
												:key="value"
												:value="value"
											>
												{{ name }}
											</option>
										</select>
										<span
											class="fw-normal text-decoration-underline text-nowrap text-muted pe-none"
										>
											{{ vehicleFilter || $t("sessions.filter.filter") }}
										</span>
									</label>
								</th>
								<th scope="col" class="align-top text-end">
									{{ $t("sessions.energy") }}
									<div>{{ fmtKWh(chargedEnergy * 1e3) }}</div>
								</th>
								<th scope="col" class="align-top text-end">
									{{ $t("sessions.solar") }}
									<div v-if="solarPercentage != null">
										{{ fmtNumber(solarPercentage, 1) }}%
									</div>
								</th>
								<th scope="col" class="align-top text-end">
									{{ $t("sessions.price") }}
									<div v-if="price != null">
										{{ fmtMoney(price, currency) }}
										{{ fmtCurrencySymbol(currency) }}
									</div>
								</th>
								<th scope="col" class="align-top text-end">
									{{ $t("sessions.avgPrice") }}
									<div v-if="pricePerKWh != null">
										{{ fmtPricePerKWh(pricePerKWh, currency) }}
									</div>
								</th>
								<th scope="col" class="align-top text-end pe-0">
									{{ $t("sessions.co2") }}
									<div v-if="co2PerKWh != null">
										{{ fmtCo2Medium(co2PerKWh) }}
									</div>
								</th>
							</tr>
						</thead>
						<tbody>
							<tr
								v-for="(session, id) in filteredSessions"
								:key="id"
								role="button"
								@click="showDetails(session.id)"
							>
								<td class="ps-0">
									{{ fmtFullDateTime(new Date(session.created), true) }}
								</td>
								<td>
									{{ session.loadpoint }}
								</td>
								<td>
									{{ session.vehicle }}
								</td>
								<td class="text-end">
									{{ fmtKWh(session.chargedEnergy * 1e3) }}
								</td>
								<td class="text-end">
									<span v-if="session.solarPercentage != null">
										{{ fmtNumber(session.solarPercentage, 1) }}%
									</span>
									<span v-else class="text-muted">-</span>
								</td>
								<td class="text-end">
									<span v-if="session.price != null">
										{{ fmtMoney(session.price, currency) }}
										{{ fmtCurrencySymbol(currency) }}
									</span>
									<span v-else class="text-muted">-</span>
								</td>
								<td class="text-end">
									<span v-if="session.pricePerKWh != null">
										{{ fmtPricePerKWh(session.pricePerKWh, currency) }}
									</span>
									<span v-else class="text-muted">-</span>
								</td>
								<td class="text-end pe-0">
									<span v-if="session.co2PerKWh != null">
										{{ fmtCo2Medium(session.co2PerKWh) }}
									</span>
									<span v-else class="text-muted">-</span>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
				<div class="d-flex mb-5">
					<a
						class="btn btn-outline-secondary text-nowrap me-3"
						:href="csvHrefLink('2023.05')"
						download="sessions.csv"
					>
						CSV Mai 2023 herunterladen
					</a>
					<a
						class="btn btn-outline-secondary text-nowrap"
						:href="csvHrefLink()"
						download="sessions.csv"
					>
						CSV Gesamt herunterladen
					</a>
				</div>
			</main>
			<ChargingSessionModal
				:session="selectedSession"
				:vehicles="vehicles"
				@session-changed="loadSessions"
			/>
		</div>
	</div>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";
import TopNavigation from "../components/TopNavigation.vue";
import "@h2d2/shopicons/es/bold/arrowback";
import "@h2d2/shopicons/es/regular/angledoubleleftsmall";
import "@h2d2/shopicons/es/regular/angledoublerightsmall";
import formatter from "../mixins/formatter";
import api from "../api";
import store from "../store";
import ChargingSessionModal from "../components/ChargingSessionModal.vue";

export default {
	name: "ChargingSessions",
	components: { TopNavigation, ChargingSessionModal },
	mixins: [formatter],
	props: {
		notifications: Array,
	},
	data() {
		return {
			sessions: [],
			selectedSessionId: undefined,
			month: 5,
			year: 2023,
			vehicleFilter: "",
			loadpointFilter: "",
		};
	},
	computed: {
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
		filteredSessions() {
			return this.currentSessions
				.filter((s) => !this.loadpointFilter || s.loadpoint === this.loadpointFilter)
				.filter((s) => !this.vehicleFilter || s.vehicle === this.vehicleFilter);
		},
		vehicleFilterOptions() {
			const options = [{ name: this.$t("sessions.filter.allVehicles"), value: "" }];
			this.vehicles.forEach((v) => {
				options.push({ name: `${v.name} (${v.count})`, value: v.name });
			});
			return options;
		},
		loadpointFilterOptions() {
			const options = [{ name: this.$t("sessions.filter.allLoadpoints"), value: "" }];
			this.loadpoints.forEach((v) => {
				options.push({ name: `${v.name} (${v.count})`, value: v.name });
			});
			return options;
		},
		chargedEnergy() {
			return this.totalKWh(this.filteredSessions);
		},
		price() {
			return this.totalPrice(this.filteredSessions);
		},
		pricePerKWh() {
			return this.price / this.chargedEnergy;
		},
		co2PerKWh() {
			return 999;
		},
		solarPercentage() {
			const chargedSolarEnergy = this.chargedSolarEnergyKWh(this.filteredSessions);
			return (100 / this.chargedEnergy) * chargedSolarEnergy;
		},
		loadpoints() {
			const result = {};
			this.currentSessions.forEach((s) => {
				if (result[s.loadpoint] === undefined) {
					result[s.loadpoint] = 0;
				}
				result[s.loadpoint]++;
			});
			return Object.entries(result).map(([name, count]) => ({ name, count }));
		},
		vehicles() {
			const result = {};
			this.currentSessions.forEach((s) => {
				if (result[s.vehicle] === undefined) {
					result[s.vehicle] = 0;
				}
				result[s.vehicle]++;
			});
			return Object.entries(result).map(([name, count]) => ({ name, count }));
		},
		selectedSession() {
			return this.sessions.find((s) => s.id == this.selectedSessionId);
		},
		currency() {
			return store.state.currency;
		},
	},
	mounted() {
		this.loadSessions();
	},
	methods: {
		changeLoadpointFilter(event) {
			this.loadpointFilter = event.target.value;
		},
		changeVehicleFilter(event) {
			this.vehicleFilter = event.target.value;
		},
		async loadSessions() {
			const response = await api.get("sessions");
			this.sessions = response.data?.result;
		},
		sessionsByMonth(sessions, month, year) {
			return sessions.filter((session) => {
				const date = new Date(session.created);
				return date.getFullYear() === year && date.getMonth() + 1 === month;
			});
		},
		groupByLoadpoint(sessions) {
			return sessions.reduce((groups, session) => {
				const loadpoint = session.loadpoint;
				if (!groups[loadpoint]) groups[loadpoint] = [];
				groups[loadpoint].push(session);
				return groups;
			}, {});
		},
		totalKWh(sessions) {
			return sessions.reduce((total, session) => total + session.chargedEnergy, 0);
		},
		chargedSolarEnergyKWh(sessions) {
			return sessions.reduce(
				(total, session) => total + session.chargedEnergy * (session.solarPercentage / 100),
				0
			);
		},
		totalPrice(sessions) {
			return sessions.reduce((total, session) => total + session.price, 0);
		},
		groupedKWh(by, sessions) {
			const grouped = sessions.reduce((groups, session) => {
				const name = session[by];
				if (!groups[name]) groups[name] = 0;
				groups[name] += session.chargedEnergy * 1e3;
				return groups;
			}, {});
			const list = Object.entries(grouped).map(([name, energy]) => {
				return { name, energy };
			});
			return list.length >= 2 ? list : [];
		},
		formatGroupHeadline(group) {
			const date = new Date();
			const [year, month] = group.split(".");
			date.setMonth(month - 1);
			date.setFullYear(year);
			return this.fmtMonthYear(date);
		},
		showDetails(sessionId) {
			this.selectedSessionId = sessionId;
			const modal = Modal.getOrCreateInstance(document.getElementById("sessionDetailsModal"));
			modal.show();
		},
		csvHrefLink(groupKey) {
			var url = `./api/sessions?format=csv&lang=${this.$i18n.locale}`;
			if (groupKey) {
				const [year, month] = groupKey.split(".");
				url += `&year=${year}`;

				if (month) {
					url += `&month=${month}`;
				}
			}
			return url;
		},
	},
};
</script>
<style scoped>
.back {
	width: 22px;
	height: 22px;
	position: relative;
	top: -2px;
}
.custom-select {
	left: 0;
	top: 0;
	bottom: 0;
	right: 0;
	position: absolute;
	opacity: 0;
	-webkit-appearance: menulist-button;
}
</style>
