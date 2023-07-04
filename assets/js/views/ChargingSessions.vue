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
				<div v-if="currentSessions.length === 0" data-testid="sessions-nodata" class="my-5">
					<p>{{ $t("sessions.noData") }}</p>
				</div>
				<div v-else class="table-responsive my-3">
					<table class="table text-nowrap">
						<thead>
							<tr data-testid="sessions-head">
								<th scope="col" class="align-top ps-0">
									{{ $t("sessions.date") }}
								</th>
								<th
									v-if="showLoadpoints"
									scope="col"
									class="align-top"
									data-testid="loadpoint"
								>
									{{ $t("sessions.loadpoint") }}
									<label class="position-relative d-block">
										<select
											:value="loadpointFilter"
											class="custom-select"
											@change="changeLoadpointFilter"
										>
											<option
												v-for="{
													name,
													value,
													count,
												} in loadpointFilterOptions"
												:key="value"
												:value="value"
												:disabled="count === 0"
											>
												{{ name }} ({{ count }})
											</option>
										</select>
										<span
											class="fw-normal text-decoration-underline text-nowrap text-muted pe-none"
										>
											{{ loadpointFilter || $t("sessions.filter.filter") }}
										</span>
									</label>
								</th>
								<th
									v-if="showVehicles"
									scope="col"
									class="align-top"
									data-testid="vehicle"
								>
									{{ $t("sessions.vehicle") }}
									<label class="position-relative d-block">
										<select
											:value="vehicleFilter"
											class="custom-select"
											@change="changeVehicleFilter"
										>
											<option
												v-for="{
													name,
													value,
													count,
												} in vehicleFilterOptions"
												:key="value"
												:value="value"
												:disabled="count === 0"
											>
												{{ name }} ({{ count }})
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
									<div class="text-muted fw-normal">
										{{ fmtKWh(chargedEnergy * 1e3, chargedEnergy >= 1) }}
									</div>
								</th>
								<th
									v-if="hasSolarPercentage"
									scope="col"
									class="align-top text-end"
								>
									{{ $t("sessions.solar") }}
									<div
										v-if="solarPercentage != null"
										class="text-muted fw-normal"
									>
										{{ fmtNumber(solarPercentage, 1) }}%
									</div>
								</th>
								<th v-if="hasPrice" scope="col" class="align-top text-end">
									{{ $t("sessions.price") }}
									<div v-if="price != null" class="text-muted fw-normal">
										{{ fmtMoney(price, currency) }}
										{{ fmtCurrencySymbol(currency) }}
									</div>
								</th>
								<th v-if="hasPrice" scope="col" class="align-top text-end">
									{{ $t("sessions.avgPrice") }}
									<div v-if="pricePerKWh != null" class="text-muted fw-normal">
										{{ fmtPricePerKWh(pricePerKWh, currency) }}
									</div>
								</th>
								<th
									v-if="hasCo2"
									scope="col"
									class="align-top text-end pe-0"
									data-testid="co2"
								>
									{{ $t("sessions.co2") }}
									<div v-if="co2PerKWh != null" class="text-muted fw-normal">
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
								data-testid="sessions-entry"
								@click="showDetails(session.id)"
							>
								<td class="ps-0">
									<u>{{ fmtFullDateTime(new Date(session.created), true) }}</u>
								</td>
								<td v-if="showLoadpoints">
									{{ session.loadpoint }}
								</td>
								<td v-if="showVehicles">
									{{ session.vehicle }}
								</td>
								<td class="text-end">
									{{
										fmtKWh(
											session.chargedEnergy * 1e3,
											session.chargedEnergy >= 1
										)
									}}
								</td>
								<td v-if="hasSolarPercentage" class="text-end">
									<span v-if="session.solarPercentage != null">
										{{ fmtNumber(session.solarPercentage, 1) }}%
									</span>
									<span v-else class="text-muted">-</span>
								</td>
								<td v-if="hasPrice" class="text-end">
									<span v-if="session.price != null">
										{{ fmtMoney(session.price, currency) }}
										{{ fmtCurrencySymbol(currency) }}
									</span>
									<span v-else class="text-muted">-</span>
								</td>
								<td v-if="hasPrice" class="text-end">
									<span v-if="session.pricePerKWh != null">
										{{ fmtPricePerKWh(session.pricePerKWh, currency) }}
									</span>
									<span v-else class="text-muted">-</span>
								</td>
								<td v-if="hasCo2" class="text-end pe-0">
									<span v-if="session.co2PerKWh != null">
										{{ fmtCo2Medium(session.co2PerKWh) }}
									</span>
									<span v-else class="text-muted">-</span>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
				<div class="d-grid gap-2 d-sm-block mt-3 mb-5">
					<a
						v-if="currentSessions.length"
						class="btn btn-outline-secondary text-nowrap me-sm-2"
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
				:vehicles="vehiclesObjects"
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
			return this.currentSessions.filter(this.filterByLoadpoint).filter(this.filterByVehicle);
		},
		vehicleFilterOptions() {
			const options = [
				{
					name: this.$t("sessions.filter.allVehicles"),
					value: "",
					count: this.filterCountForVehicle(),
				},
			];
			this.vehicles.forEach((name) => {
				const count = this.filterCountForVehicle(name);
				options.push({ name, value: name, count });
			});
			return options;
		},
		loadpointFilterOptions() {
			const options = [
				{
					name: this.$t("sessions.filter.allLoadpoints"),
					value: "",
					count: this.filterCountForLoadpoint(),
				},
			];
			this.loadpoints.forEach((name) => {
				const count = this.filterCountForLoadpoint(name);
				options.push({ name, value: name, count });
			});
			return options;
		},
		chargedEnergy() {
			return this.filteredSessions.reduce((total, s) => total + s.chargedEnergy, 0);
		},
		price() {
			return this.filteredSessions.reduce((total, s) => total + s.price, 0);
		},
		hasPrice() {
			return this.filteredSessions.find((s) => s.price != null) != null;
		},
		hasSolarPercentage() {
			return this.filteredSessions.find((s) => s.solarPercentage != null) != null;
		},
		hasCo2() {
			return this.filteredSessions.find((s) => s.co2PerKWh != null) != null;
		},
		showVehicles() {
			return this.hasMultipleVehicles || this.vehicleFilter;
		},
		showLoadpoints() {
			return this.hasMultipleLoadpoints || this.loadpointFilter;
		},
		hasMultipleVehicles() {
			const vehicles = this.currentSessions.map((s) => s.vehicle);
			return new Set(vehicles).size > 1;
		},
		hasMultipleLoadpoints() {
			const loadpoints = this.currentSessions.map((s) => s.loadpoint);
			return new Set(loadpoints).size > 1;
		},
		pricePerKWh() {
			return this.price / this.chargedEnergy;
		},
		co2PerKWh() {
			const emittedCo2 = this.filteredSessions.reduce(
				(total, s) => total + s.chargedEnergy * s.co2PerKWh,
				0
			);
			return emittedCo2 / this.chargedEnergy;
		},
		solarPercentage() {
			const chargedSolarEnergy = this.filteredSessions.reduce(
				(total, s) => total + s.chargedEnergy * (s.solarPercentage / 100),
				0
			);
			return (100 / this.chargedEnergy) * chargedSolarEnergy;
		},
		loadpoints() {
			return [...new Set(this.currentSessions.map((s) => s.loadpoint))];
		},
		vehicles() {
			return [...new Set(this.currentSessions.map((s) => s.vehicle))];
		},
		vehiclesObjects() {
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
		headline() {
			const date = new Date();
			date.setMonth(this.month - 1);
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
			date.setMonth(this.month - 2);
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
			date.setMonth(this.month);
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
		filterByLoadpoint(session) {
			return !this.loadpointFilter || session.loadpoint === this.loadpointFilter;
		},
		filterByVehicle(session) {
			return !this.vehicleFilter || session.vehicle === this.vehicleFilter;
		},
		filterCountForVehicle(vehicle) {
			return this.currentSessions
				.filter(this.filterByLoadpoint)
				.filter((s) => !vehicle || s.vehicle === vehicle).length;
		},
		filterCountForLoadpoint(loadpoint) {
			return this.currentSessions
				.filter(this.filterByVehicle)
				.filter((s) => !loadpoint || s.loadpoint === loadpoint).length;
		},
		changeLoadpointFilter(event) {
			const loadpoint = event.target.value || undefined;
			this.$router.push({ query: { ...this.$route.query, loadpoint } });
		},
		changeVehicleFilter(event) {
			const vehicle = event.target.value || undefined;
			this.$router.push({ query: { ...this.$route.query, vehicle } });
		},
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
				lang: this.$i18n.locale,
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
