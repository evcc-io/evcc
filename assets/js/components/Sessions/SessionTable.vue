<template>
	<h3 class="fw-normal mb-4">{{ $t("sessions.overview") }}</h3>

	<div v-if="sessions.length === 0" data-testid="sessions-nodata" class="mb-5">
		<p>{{ $t("sessions.noData") }}</p>
	</div>
	<div v-else class="mb-5 table-outer">
		<table class="table text-nowrap">
			<thead class="sticky-top">
				<tr data-testid="sessions-head">
					<th scope="col" class="align-top ps-0">
						{{ $t("sessions.date") }}
					</th>
					<th
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
					<th scope="col" class="align-top d-none d-md-table-cell" data-testid="vehicle">
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
						<div class="d-flex flex-wrap text-truncate">
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
									{{ loadpointFilter || $t("sessions.filter.filter") }}
								</span>
							</CustomSelect>
						</div>
						<div class="text-truncate d-flex flex-wrap">
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
						v-for="(column, index) in columnsPerBreakpoint"
						:key="column.name"
						scope="col"
						:data-testid="`sessions-head-${column.name}`"
						class="align-top text-end"
					>
						<CustomSelect
							v-if="tooMuchColumns"
							:selected="column.name"
							:options="columnOptions"
							data-testid="column"
							@change="selectColumnPosition(index, $event.target.value)"
						>
							<span class="text-decoration-underline">
								{{ $t(`sessions.${column.name}`) }}
							</span>
						</CustomSelect>
						<span v-else>
							{{ $t(`sessions.${column.name}`) }}
						</span>
						<div class="text-gray fw-normal">{{ column.unit }}</div>
					</th>
				</tr>
			</thead>
			<tfoot class="sticky-bottom">
				<tr data-testid="sessions-foot">
					<th scope="col" class="align-top ps-0">
						{{ $t("sessions.total") }}
					</th>
					<th scope="col" class="d-none d-md-table-cell"></th>
					<th scope="col" class="d-none d-md-table-cell"></th>
					<th scope="col" class="d-md-none"></th>
					<th
						v-for="column in columnsPerBreakpoint"
						:key="column.name"
						:data-testid="`sessions-foot-${column.name}`"
						scope="col"
						class="align-top text-end"
					>
						{{ column.format(column.total || 0) }}
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
					<td class="d-none d-md-table-cell">
						{{ session.loadpoint }}
					</td>
					<td class="d-none d-md-table-cell">
						{{ session.vehicle }}
					</td>
					<td class="d-md-none text-truncate">
						<div>{{ session.loadpoint }}</div>
						<div>{{ session.vehicle }}</div>
					</td>
					<td v-for="column in columnsPerBreakpoint" :key="column.name" class="text-end">
						<span v-if="column.value(session) === null" class="text-gray"> - </span>
						<span v-else>{{ column.format(column.value(session) || 0) }}</span>
					</td>
				</tr>
			</tbody>
		</table>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import CustomSelect from "../Helper/CustomSelect.vue";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import breakpoint from "@/mixins/breakpoint.ts";
import settings from "@/settings";
import type { CURRENCY } from "@/types/evcc";
import type { Session, Column } from "./types";

const COLUMNS_PER_BREAKPOINT = {
	xs: 1,
	sm: 2,
	md: 3,
	lg: 4,
	xl: 5,
	xxl: 6,
};

export default defineComponent({
	name: "SessionTable",
	components: { CustomSelect },
	mixins: [formatter, breakpoint],
	props: {
		sessions: { type: Array as PropType<Session[]>, default: () => [] },
		loadpointFilter: { type: String, default: "" },
		vehicleFilter: { type: String, default: "" },
		currency: { type: String as PropType<CURRENCY> },
	},
	emits: ["show-session"],
	data() {
		return {
			selectedColumns: settings.sessionColumns,
		};
	},
	computed: {
		filteredSessions() {
			return this.sessions
				.filter(this.filterByLoadpoint)
				.filter(this.filterByVehicle)
				.sort((a, b) => new Date(b.created).getTime() - new Date(a.created).getTime());
		},
		maxColumns() {
			return COLUMNS_PER_BREAKPOINT[this.breakpoint] || 1;
		},
		columns() {
			const columns: Column[] = [
				{
					name: "energy",
					unit: "kWh",
					total: this.chargedEnergy,
					value: (session) => session.chargedEnergy,
					format: (value) => this.fmtWh(value * 1e3, POWER_UNIT.KW, false),
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
					value: (session) => session.co2PerKWh || null,
					format: (value) => this.fmtNumber(value, 0),
				},
				{
					name: "chargeDuration",
					unit: "h:mm",
					total: this.chargeDuration,
					value: (session) => session.chargeDuration,
					format: (value) => this.fmtDurationNs(value, false, "h"),
				},
				{
					name: "avgPower",
					unit: "kW",
					total: this.avgPower,
					value: (session) => {
						if (session.chargedEnergy && session.chargeDuration) {
							return session.chargedEnergy / this.nsToHours(session.chargeDuration);
						}
						return null;
					},
					format: (value) =>
						value ? this.fmtW(value * 1e3, POWER_UNIT.KW, false, 1) : undefined,
				},
			];
			// only columns with values are shown
			return columns.filter((column) => {
				if (column.name === "energy") return true;
				return this.sessions.some((s) => column.value(s));
			});
		},
		tooMuchColumns() {
			return this.columns.length > this.maxColumns;
		},
		sortedColumns() {
			const columns = [...this.columns];
			const sorted = [] as Column[];
			for (const name of this.selectedColumns) {
				if (!name && columns.length) {
					sorted.push(columns.shift() as Column);
				} else if (columns.some((c) => c.name === name)) {
					const column = columns.find((c) => c.name === name);
					if (column) {
						sorted.push(column);
						const index = columns.indexOf(column);
						columns.splice(index, 1);
					}
				}
			}
			return sorted.concat(columns);
		},
		columnsPerBreakpoint() {
			return this.sortedColumns.slice(0, this.maxColumns);
		},
		columnOptions() {
			return this.columns.map((column) => {
				return {
					name: this.$t(`sessions.${column.name}`),
					value: column.name,
					disabled: this.columnsPerBreakpoint.some((c) => c.name === column.name),
				};
			});
		},
		vehicleFilterOptions() {
			const options = [
				{
					name: this.$t("sessions.filter.allVehicles"),
					value: "",
					count: this.filterCountForVehicle(""),
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
					count: this.filterCountForLoadpoint(""),
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
		chargeDuration() {
			return this.filteredSessions.reduce((total, s) => total + s.chargeDuration, 0);
		},
		price() {
			return this.filteredSessions.reduce((total, s) => total + (s.price || 0), 0);
		},
		avgPower() {
			const { energy, hours } = this.filteredSessions
				.filter((s) => s.chargedEnergy && s.chargeDuration)
				.reduce(
					(total, s) => {
						total.energy += s.chargedEnergy;
						total.hours += this.nsToHours(s.chargeDuration);
						return total;
					},
					{ energy: 0, hours: 0 }
				);
			if (energy && hours) {
				return energy / hours;
			}
			return null;
		},
		pricePerKWh() {
			const total = this.filteredSessions
				.filter((s) => s.price !== null)
				.reduce(
					(total, s) => ({
						price: total.price + (s.price || 0),
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
						emittedCo2: total.emittedCo2 + s.chargedEnergy * (s.co2PerKWh || 0),
						chargedEnergy: total.chargedEnergy + s.chargedEnergy,
					}),
					{ emittedCo2: 0, chargedEnergy: 0 }
				);
			if (total.chargedEnergy && total.emittedCo2) {
				return total.emittedCo2 / total.chargedEnergy;
			}
			return null;
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
			return [...new Set(this.sessions.map((s) => s.loadpoint))];
		},
		vehicles() {
			return [...new Set(this.sessions.map((s) => s.vehicle))];
		},
	},
	methods: {
		nsToHours(ns: number) {
			return ns / 1e9 / 3600;
		},
		filterByLoadpoint(session: Session) {
			return !this.loadpointFilter || session.loadpoint === this.loadpointFilter;
		},
		filterByVehicle(session: Session) {
			return !this.vehicleFilter || session.vehicle === this.vehicleFilter;
		},
		filterCountForVehicle(vehicle: string) {
			return this.sessions
				.filter(this.filterByLoadpoint)
				.filter((s) => !vehicle || s.vehicle === vehicle).length;
		},
		filterCountForLoadpoint(loadpoint: string) {
			return this.sessions
				.filter(this.filterByVehicle)
				.filter((s) => !loadpoint || s.loadpoint === loadpoint).length;
		},
		selectColumnPosition(index: number, value: Column) {
			this.selectedColumns[index] = value;
			settings.sessionColumns = [...this.selectedColumns];
		},
		changeLoadpointFilter(event: Event) {
			const loadpoint = (event.target as HTMLSelectElement).value || undefined;
			this.$router.push({ query: { ...this.$route.query, loadpoint } });
		},
		changeVehicleFilter(event: Event) {
			const vehicle = (event.target as HTMLSelectElement).value || undefined;
			this.$router.push({ query: { ...this.$route.query, vehicle } });
		},
		showDetails(sessionId: number) {
			this.$emit("show-session", sessionId);
		},
	},
});
</script>
<style scoped>
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
.sticky-top {
	top: 7rem;
}
@media (min-width: 992px) {
	.sticky-top {
		top: 4.5rem;
	}
}
.sticky-top th {
	padding-top: max(0.5rem, env(safe-area-inset-top));
}
.table-outer {
	position: relative;
	top: calc(max(0.5rem, env(safe-area-inset-top)) * -1);
}
.month-header {
	position: relative;
	z-index: 2;
}
.sticky-bottom th {
	padding-bottom: max(0.5rem, env(safe-area-inset-bottom));
	border-bottom: none;
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
