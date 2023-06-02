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

				<div v-for="loadpoint in sessionsByLoadpoint" :key="loadpoint.name">
					<div class="d-flex align-items-baseline mb-3">
						<h3 class="me-4 mb-0">
							{{ loadpoint.name }}
						</h3>
					</div>

					<div>
						<router-link to="/" class="me-2 text-muted">Alle Fahrzeuge</router-link>
						<router-link
							v-for="(vehicle, id) in groupedKWh('vehicle', loadpoint.sessions)"
							:key="id"
							class="me-2 text-muted text-decoration-none"
							to="/"
							>{{ vehicle.name }}</router-link
						>
					</div>

					<div class="table-responsive my-3">
						<table class="table text-nowrap">
							<thead>
								<tr>
									<th scope="col" class="ps-0">{{ $t("sessions.date") }}</th>
									<th scope="col">{{ $t("sessions.vehicle") }}</th>
									<th scope="col" class="text-end">
										{{ $t("sessions.energy") }}
									</th>
									<th scope="col" class="text-end">
										{{ $t("sessions.solar") }}
									</th>
									<th scope="col" class="text-end">
										{{ $t("sessions.price") }}
									</th>
									<th scope="col" class="text-end">
										{{ $t("sessions.avgPrice") }}
									</th>
									<th scope="col" class="text-end pe-0">
										{{ $t("sessions.co2") }}
									</th>
								</tr>
							</thead>
							<tbody>
								<tr
									v-for="(session, id) in loadpoint.sessions"
									:key="id"
									role="button"
									@click="showDetails(session.id)"
								>
									<td class="ps-0">
										{{ fmtFullDateTime(new Date(session.created), true) }}
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
							<tfoot>
								<tr>
									<th class="ps-0">Gesamt</th>
									<th></th>
									<th class="text-end">
										{{ fmtKWh(loadpoint.chargedEnergy * 1e3) }}
									</th>
									<th class="text-end">
										<span v-if="loadpoint.solarPercentage != null">
											{{ fmtNumber(loadpoint.solarPercentage, 1) }}%
										</span>
										<span v-else class="text-muted">-</span>
									</th>
									<th class="text-end">
										<span v-if="loadpoint.price != null">
											{{ fmtMoney(loadpoint.price, currency) }}
											{{ fmtCurrencySymbol(currency) }}
										</span>
										<span v-else class="text-muted">-</span>
									</th>
									<th class="text-end">
										<span v-if="loadpoint.pricePerKWh != null">
											{{ fmtPricePerKWh(loadpoint.pricePerKWh, currency) }}
										</span>
										<span v-else class="text-muted">-</span>
									</th>
									<th class="text-end pe-0">
										<span v-if="loadpoint.co2PerKWh != null">
											{{ fmtCo2Medium(loadpoint.co2PerKWh) }}
										</span>
										<span v-else class="text-muted">-</span>
									</th>
								</tr>
							</tfoot>
						</table>
					</div>
				</div>
				<!--</div>-->
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
		return { sessions: [], selectedSessionId: undefined };
	},
	computed: {
		sessionsByLoadpoint() {
			const sessionsWithDefaults = this.sessions.map((session) => {
				const loadpoint = session.loadpoint || this.$t("main.loadpoint.fallbackName");
				const vehicle = session.vehicle || this.$t("main.vehicle.unknown");
				return { ...session, loadpoint, vehicle };
			});

			const sessions = this.sessionsByMonth(sessionsWithDefaults, 5, 2023);

			return Object.entries(this.groupByLoadpoint(sessions)).map(
				([loadpoint, sessionsByLoadpoint]) => {
					const chargedEnergy = this.totalKWh(sessionsByLoadpoint);
					const chargedSolarEnergy = this.chargedSolarEnergyKWh(sessionsByLoadpoint);
					const price = this.totalPrice(sessionsByLoadpoint);
					const pricePerKWh = price / chargedEnergy;
					console.log(chargedEnergy, chargedSolarEnergy);
					const solarPercentage = (100 / chargedEnergy) * chargedSolarEnergy;
					return {
						name: loadpoint,
						chargedEnergy,
						price,
						pricePerKWh,
						solarPercentage,
						sessions: sessionsByLoadpoint,
					};
				}
			);
		},
		vehicles() {
			return (
				store.state.vehicles?.map((v, index) => {
					return { id: index, title: v };
				}) || []
			);
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
.breakdown {
	list-style: none;
}

.breakdown-item {
	white-space: nowrap;
}

/* breakpoint sm */
@media (min-width: 576px) {
	.breakdown-item:after {
		content: ", ";
		white-space: wrap;
		margin-right: 0.25rem;
	}
	.breakdown-item:last-child:after {
		content: "";
	}
}

.breakdown:empty {
	display: none;
}
</style>
