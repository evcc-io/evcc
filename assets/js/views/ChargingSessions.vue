<template>
	<div class="container px-4">
		<header class="d-flex justify-content-between align-items-center py-3">
			<h1 class="mb-1 pt-1 d-flex text-nowrap text-truncate">
				<router-link class="evcc-default-text" to="/">
					<shopicon-regular-home size="s" class="home"></shopicon-regular-home>
				</router-link>
				<div size="s" class="mx-2 flex-grow-0 flex-shrink-0 fw-normal">/</div>
				<span class="text-truncate">{{ $t("sessions.title") }}</span>
			</h1>
			<TopNavigation />
		</header>

		<div class="row">
			<main class="col-12">
				<div class="d-flex align-items-baseline justify-content-between my-3 my-md-5">
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
				<div v-else class="my-3 my-md-5 table-outer">
					<table class="table text-nowrap">
						<thead class="sticky-top">
							<tr data-testid="sessions-head">
								<th scope="col" class="align-top ps-0">
									{{ $t("sessions.date") }}
								</th>
								<th
									v-if="showLoadpoints"
									scope="col"
									class="align-top d-none d-md-table-cell"
									data-testid="loadpoint"
								>
									{{ $t("sessions.loadpoint") }}
									<CustomSelect
										:selected="loadpointFilter"
										:options="loadpointFilterOptions"
										data-testid="filter-loadpoint"
										@change="changeLoadpointFilter"
									>
										<span
											class="fw-normal text-decoration-underline text-nowrap text-gray pe-none"
										>
											{{ loadpointFilter || $t("sessions.filter.filter") }}
										</span>
									</CustomSelect>
								</th>
								<th
									v-if="showVehicles"
									scope="col"
									class="align-top d-none d-md-table-cell"
									data-testid="vehicle"
								>
									{{ $t("sessions.vehicle") }}
									<CustomSelect
										:selected="vehicleFilter"
										:options="vehicleFilterOptions"
										data-testid="filter-vehicle"
										@change="changeVehicleFilter"
									>
										<span
											class="fw-normal text-decoration-underline text-nowrap text-gray pe-none"
										>
											{{ vehicleFilter || $t("sessions.filter.filter") }}
										</span>
									</CustomSelect>
								</th>
								<th scope="col" class="align-top d-md-none text-truncate">
									<div
										v-if="showLoadpoints"
										class="d-flex flex-wrap text-truncate"
									>
										<div class="me-2 text-truncate">
											{{ $t("sessions.loadpoint") }}
										</div>
										<CustomSelect
											:selected="loadpointFilter"
											:options="loadpointFilterOptions"
											data-testid="filter-loadpoint"
											@change="changeLoadpointFilter"
										>
											<span
												class="fw-normal text-decoration-underline text-nowrap text-gray pe-none text-truncate"
											>
												{{
													loadpointFilter || $t("sessions.filter.filter")
												}}
											</span>
										</CustomSelect>
									</div>
									<div
										class="text-truncate"
										:class="{ 'd-flex flex-wrap': showLoadpoints }"
									>
										<div class="me-2 text-truncate">
											{{ $t("sessions.vehicle") }}
										</div>
										<CustomSelect
											:selected="vehicleFilter"
											:options="vehicleFilterOptions"
											data-testid="filter-vehicle"
											@change="changeVehicleFilter"
										>
											<span
												class="fw-normal text-decoration-underline text-nowrap text-gray pe-none text-truncate"
											>
												{{ vehicleFilter || $t("sessions.filter.filter") }}
											</span>
										</CustomSelect>
									</div>
								</th>
								<th
									v-for="column in columns"
									:key="column.name"
									scope="col"
									:data-testid="`sessions-head-${column.name}`"
									class="align-top text-end d-none d-md-table-cell"
								>
									{{ $t(`sessions.${column.name}`) }}<br />
									<div class="text-gray fw-normal">{{ column.unit }}</div>
								</th>
								<th
									scope="col"
									class="align-top text-end d-md-none"
									data-testid="sessions-head-mobile"
								>
									<span v-if="columns.length === 1">
										{{ $t(`sessions.${mobileColumn.name}`) }}
									</span>
									<CustomSelect
										v-else
										:selected="selectedMobileColumn"
										:options="mobileColumnOptions"
										data-testid="mobile-column"
										@change="changeMobileColumn"
									>
										<span class="text-decoration-underline">
											{{ $t(`sessions.${mobileColumn.name}`) }}
										</span>
									</CustomSelect>
									<div class="text-gray fw-normal">{{ mobileColumn.unit }}</div>
								</th>
							</tr>
						</thead>
						<tfoot class="sticky-bottom">
							<tr data-testid="sessions-foot">
								<th scope="col" class="align-top ps-0">
									{{ $t("sessions.total") }}
								</th>
								<th
									v-if="showLoadpoints"
									scope="col"
									class="d-none d-md-table-cell"
								></th>
								<th
									v-if="showVehicles"
									scope="col"
									class="d-none d-md-table-cell"
								></th>
								<th scope="col" class="d-md-none"></th>
								<th
									v-for="column in columns"
									:key="column.name"
									:data-testid="`sessions-foot-${column.name}`"
									scope="col"
									class="align-top text-end d-none d-md-table-cell"
								>
									{{ column.format(column.total) }}
								</th>
								<th
									scope="col"
									data-testid="sessions-foot-mobile"
									class="align-top text-end d-md-none"
								>
									{{ mobileColumn.format(mobileColumn.total) }}
								</th>
							</tr>
						</tfoot>
						<tbody>
							<tr
								v-for="(session, id) in filteredSessions"
								:key="id"
								role="button"
								data-testid="sessions-entry"
								@click="showDetails(session.id)"
							>
								<td class="ps-0">
									{{ fmtFullDateTime(new Date(session.created), true) }}
								</td>
								<td v-if="showLoadpoints" class="d-none d-md-table-cell">
									{{ session.loadpoint }}
								</td>
								<td v-if="showVehicles" class="d-none d-md-table-cell">
									{{ session.vehicle }}
								</td>
								<td class="d-md-none text-truncate">
									<div v-if="showLoadpoints">{{ session.loadpoint }}</div>
									<div>{{ session.vehicle }}</div>
								</td>
								<td
									v-for="column in columns"
									:key="column.name"
									class="text-end d-none d-md-table-cell"
								>
									<span v-if="column.value(session) === null" class="text-gray">
										-
									</span>
									<span v-else>{{ column.format(column.value(session)) }}</span>
								</td>
								<td class="text-end d-md-none">
									<span
										v-if="mobileColumn.value(session) === null"
										class="text-gray"
									>
										-
									</span>
									<span v-else>{{
										mobileColumn.format(mobileColumn.value(session))
									}}</span>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
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
				:vehicles="vehiclesObjects"
				@session-changed="loadSessions"
			/>
		</div>
	</div>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";
import TopNavigation from "../components/TopNavigation.vue";
import "@h2d2/shopicons/es/regular/home";
import "@h2d2/shopicons/es/regular/angledoubleleftsmall";
import "@h2d2/shopicons/es/regular/angledoublerightsmall";
import formatter from "../mixins/formatter";
import api from "../api";
import store from "../store";
import CustomSelect from "../components/CustomSelect.vue";
import ChargingSessionModal from "../components/ChargingSessionModal.vue";

export default {
	name: "ChargingSessions",
	components: { TopNavigation, ChargingSessionModal, CustomSelect },
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
			selectedMobileColumn: undefined,
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
		columns() {
			const columns = [
				{
					name: "energy",
					unit: "kWh",
					total: this.chargedEnergy,
					value: (session) => session.chargedEnergy,
					format: (value) => this.fmtKWh(value * 1e3, true, false),
				},
				{
					name: "solar",
					unit: "%",
					total: this.solarPercentage,
					value: (session) => session.solarPercentage,
					format: (value) => this.fmtNumber(value, 1),
				},
				{
					name: "price",
					unit: this.fmtCurrencySymbol(this.currency),
					total: this.price,
					value: (session) => session.price,
					format: (value) => this.fmtMoney(value, this.currency),
				},
				{
					name: "avgPrice",
					unit: this.pricePerKWhUnit(this.currency),
					total: this.pricePerKWh,
					value: (session) => session.pricePerKWh,
					format: (value) => this.fmtPricePerKWh(value, this.currency, false, false),
				},
				{
					name: "co2",
					unit: "g/kWh",
					total: this.co2PerKWh,
					value: (session) => session.co2PerKWh,
					format: (value) => this.fmtNumber(value, 0),
				},
			];
			// only columns with values are shown
			return columns.filter((column) => {
				if (column.name === "energy") return true;
				return this.currentSessions.some((s) => column.value(s));
			});
		},
		mobileColumn() {
			const column = this.columns.find((column) => column.name === this.selectedMobileColumn);
			return column || this.columns[0];
		},
		mobileColumnOptions() {
			return this.columns.map((column) => {
				return {
					name: this.$t(`sessions.${column.name}`),
					value: column.name,
				};
			});
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
			const total = this.filteredSessions
				.filter((s) => s.price !== null)
				.reduce(
					(total, s) => ({
						price: total.price + s.price,
						chargedEnergy: total.chargedEnergy + s.chargedEnergy,
					}),
					{ price: 0, chargedEnergy: 0 }
				);
			return total.price / total.chargedEnergy;
		},
		co2PerKWh() {
			const total = this.filteredSessions
				.filter((s) => s.co2PerKWh !== null)
				.reduce(
					(total, s) => ({
						emittedCo2: total.emittedCo2 + s.chargedEnergy * s.co2PerKWh,
						chargedEnergy: total.chargedEnergy + s.chargedEnergy,
					}),
					{ emittedCo2: 0, chargedEnergy: 0 }
				);
			return total.emittedCo2 / total.chargedEnergy;
		},
		solarPercentage() {
			const total = this.filteredSessions
				.filter((s) => s.solarPercentage !== null)
				.reduce(
					(total, s) => ({
						chargedSolarEnergy:
							total.chargedSolarEnergy + s.chargedEnergy * (s.solarPercentage / 100),
						chargedEnergy: total.chargedEnergy + s.chargedEnergy,
					}),
					{ chargedSolarEnergy: 0, chargedEnergy: 0 }
				);

			return (100 / total.chargedEnergy) * total.chargedSolarEnergy;
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
		changeMobileColumn(event) {
			this.selectedMobileColumn = event.target.value;
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
.home {
	height: 22px;
	width: 22px;
	position: relative;
	top: -3px;
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
.table {
	border-collapse: separate;
	border-spacing: 0;
}
.table thead,
.table tfoot {
	background: var(--evcc-background);
}
.table tfoot th {
	border-top-width: 2px;
}
.table thead th {
	border-bottom-width: 2px;
}
.table tbody tr:last-child td {
	border-bottom-width: 0;
}

.sticky-top,
.sticky-bottom {
	z-index: 1;
}
@media (max-width: 576px) {
	.table td,
	.table th {
		width: 50%;
	}
	.table td:first-child,
	.table th:first-child,
	.table td:last-child,
	.table th:last-child {
		width: 25%;
	}

	.table td.text-truncate,
	.table th.text-truncate {
		max-width: 1px;
	}
}
</style>
