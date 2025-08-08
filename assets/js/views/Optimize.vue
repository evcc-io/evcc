<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader title="Optimize" />
		<div class="alert alert-light mb-4">This page is work in progress.</div>
		<div class="row">
			<main class="col-12">
				<div v-if="evopt">
					<!-- Optimization Status -->
					<div class="mb-4">
						<div class="row">
							<div class="col-md-6">
								<div class="card border-0 bg-light">
									<div class="card-body">
										<h5 class="card-title">Optimization Status</h5>
										<p class="card-text mb-1">
											<span class="badge" :class="statusBadgeClass">
												{{ evopt.res.status }}
											</span>
										</p>
										<p
											v-if="evopt.res.objective_value !== null"
											class="card-text mb-0"
										>
											<small class="text-muted">
												Economic Benefit:
												{{
													fmtMoney(
														evopt.res.objective_value,
														currency,
														true,
														true
													)
												}}
											</small>
										</p>
									</div>
								</div>
							</div>
							<div class="col-md-6">
								<div class="card border-0 bg-light">
									<div class="card-body">
										<h5 class="card-title">Optimization Parameters</h5>
										<p class="card-text mb-1">
											<small class="text-muted">
												Charging Efficiency:
												{{
													fmtPercentage(
														(evopt.req.eta_c || 0.95) * 100,
														1
													)
												}}
											</small>
										</p>
										<p class="card-text mb-1">
											<small class="text-muted">
												Discharging Efficiency:
												{{
													fmtPercentage(
														(evopt.req.eta_d || 0.95) * 100,
														1
													)
												}}
											</small>
										</p>
										<p v-if="evopt.req.M" class="card-text mb-0">
											<small class="text-muted">
												MILP Big M: {{ fmtNumber(evopt.req.M, 0) }}
											</small>
										</p>
									</div>
								</div>
							</div>
						</div>
					</div>

					<div class="mb-4">
						<h3 class="fw-normal mb-3">Battery Configuration</h3>
						<div class="table-responsive">
							<table class="table">
								<thead>
									<tr>
										<th scope="col">#</th>
										<th scope="col">Power Range (kW)</th>
										<th scope="col">Max Discharge (kW)</th>
										<th scope="col">SoC Range (kWh)</th>
										<th scope="col">Initial SoC (kWh)</th>
										<th scope="col">Energy Value (€/kWh)</th>
										<th scope="col">Grid Interaction</th>
										<th scope="col">Demand Profile</th>
										<th scope="col">SoC Goals</th>
									</tr>
								</thead>
								<tbody>
									<tr
										v-for="(battery, index) in evopt.req.batteries"
										:key="index"
									>
										<th scope="row">{{ index + 1 }}</th>
										<td>
											{{ formatPowerRange(battery.c_min, battery.c_max) }}
										</td>
										<td>{{ formatPower(battery.d_max) }}</td>
										<td>
											{{ formatEnergyRange(battery.s_min, battery.s_max) }}
										</td>
										<td>{{ formatEnergy(battery.s_initial) }}</td>
										<td>
											{{ fmtMoney(battery.p_a * 1000, currency, true, true) }}
										</td>
										<td>
											<div class="d-flex flex-column gap-1">
												<span
													v-if="battery.charge_from_grid"
													class="badge bg-primary"
												>
													Grid Charge
												</span>
												<span
													v-if="battery.discharge_to_grid"
													class="badge bg-success"
												>
													Grid Discharge
												</span>
												<span
													v-if="
														!battery.charge_from_grid &&
														!battery.discharge_to_grid
													"
													class="text-muted"
												>
													No Grid Interaction
												</span>
											</div>
										</td>
										<td>
											<span
												v-if="battery.p_demand?.length"
												class="badge bg-info"
											>
												{{ battery.p_demand.length }} steps
											</span>
											<span v-else class="text-muted">None</span>
										</td>
										<td>
											<span
												v-if="battery.s_goal?.length"
												class="badge bg-warning"
											>
												{{ battery.s_goal.length }} goals
											</span>
											<span v-else class="text-muted">None</span>
										</td>
									</tr>
								</tbody>
							</table>
						</div>
					</div>

					<div class="mb-4">
						<h3 class="fw-normal mb-3">Time Series Data</h3>
						<div class="table-responsive">
							<table class="table table-sm">
								<thead>
									<tr>
										<th scope="col" class="text-nowrap">Data Series</th>
										<th
											v-for="(_, index) in timeSlots"
											:key="index"
											scope="col"
											:class="['text-end']"
										>
											{{ formatHour(index) }}
										</th>
									</tr>
								</thead>
								<tbody>
									<!-- Request Data -->
									<tr class="table-info">
										<td colspan="100%" class="fw-bold text-start">
											Request Data
										</td>
									</tr>
									<tr>
										<td class="fw-medium text-nowrap text-start">
											Solar Forecast (kW)
										</td>
										<td
											v-for="(value, index) in evopt.req.time_series.ft"
											:key="index"
											:class="['text-end', { 'text-muted': value === 0 }]"
										>
											{{ formatPower(value) }}
										</td>
									</tr>

									<tr>
										<td class="fw-medium text-nowrap text-start">
											{{ gridFeedinPriceLabel }}
										</td>
										<td
											v-for="(value, index) in evopt.req.time_series.p_E"
											:key="index"
											:class="['text-end', { 'text-muted': value === 0 }]"
										>
											{{
												fmtPricePerKWh(value * 1000, currency, false, false)
											}}
										</td>
									</tr>
									<tr>
										<td class="fw-medium text-nowrap text-start">
											{{ gridImportPriceLabel }}
										</td>
										<td
											v-for="(value, index) in evopt.req.time_series.p_N"
											:key="index"
											:class="['text-end', { 'text-muted': value === 0 }]"
										>
											{{
												fmtPricePerKWh(value * 1000, currency, false, false)
											}}
										</td>
									</tr>
									<tr>
										<td class="fw-medium text-nowrap text-start">
											Household Demand (kW)
										</td>
										<td
											v-for="(value, index) in evopt.req.time_series.gt"
											:key="index"
											:class="['text-end', { 'text-muted': value === 0 }]"
										>
											{{ formatPower(value) }}
										</td>
									</tr>
									<tr>
										<td class="fw-medium text-nowrap text-start">
											Time Step Duration (h)
										</td>
										<td
											v-for="(value, index) in evopt.req.time_series.dt"
											:key="index"
											:class="['text-end']"
										>
											{{ formatDuration(value) }}
										</td>
									</tr>

									<!-- Response Data -->
									<tr class="table-success">
										<td colspan="100%" class="fw-bold text-start">
											Response Data
										</td>
									</tr>
									<tr>
										<td class="fw-medium text-nowrap text-start">
											Grid Export (kW)
										</td>
										<td
											v-for="(value, index) in evopt.res.grid_export"
											:key="index"
											:class="['text-end', { 'text-muted': value === 0 }]"
										>
											{{ formatPower(value) }}
										</td>
									</tr>
									<tr>
										<td class="fw-medium text-nowrap text-start">
											Grid Import (kW)
										</td>
										<td
											v-for="(value, index) in evopt.res.grid_import"
											:key="index"
											:class="['text-end', { 'text-muted': value === 0 }]"
										>
											{{ formatPower(value) }}
										</td>
									</tr>
									<tr>
										<td class="fw-medium text-nowrap text-start">
											⬇ Import / ⬆ Export
										</td>
										<td
											v-for="(value, index) in evopt.res.flow_direction"
											:key="index"
											:class="['text-end']"
										>
											<span
												:title="
													value === 1
														? 'Export to Grid'
														: 'Import from Grid'
												"
											>
												{{ value === 1 ? "⬆" : "⬇" }}
											</span>
										</td>
									</tr>

									<!-- Battery Response Data -->
									<template
										v-for="(battery, batteryIndex) in evopt.res.batteries"
										:key="batteryIndex"
									>
										<tr class="table-warning">
											<td colspan="100%" class="fw-bold text-start">
												Battery {{ batteryIndex + 1 }} Response
											</td>
										</tr>
										<tr>
											<td class="fw-medium text-nowrap text-start">
												Charging Power (kW)
											</td>
											<td
												v-for="(value, index) in battery.charging_power"
												:key="index"
												:class="['text-end', { 'text-muted': value === 0 }]"
											>
												{{ formatPower(value) }}
											</td>
										</tr>
										<tr>
											<td class="fw-medium text-nowrap text-start">
												Discharging Power (kW)
											</td>
											<td
												v-for="(value, index) in battery.discharging_power"
												:key="index"
												:class="['text-end', { 'text-muted': value === 0 }]"
											>
												{{ formatPower(value) }}
											</td>
										</tr>
										<tr>
											<td class="fw-medium text-nowrap text-start">
												State of Charge (kWh)
											</td>
											<td
												v-for="(value, index) in battery.state_of_charge"
												:key="index"
												:class="['text-end', { 'text-muted': value === 0 }]"
											>
												{{ formatEnergy(value) }}
											</td>
										</tr>
									</template>
								</tbody>
							</table>
						</div>
					</div>

					<details class="mb-4">
						<summary class="btn btn-link text-decoration-none p-0 mb-3">
							<h3 class="fw-normal d-inline text-muted text-decoration-underline">
								Raw Data
							</h3>
						</summary>
						<div>
							<p>Request:</p>
							<pre>{{ JSON.stringify(evopt.req, null, 2) }}</pre>
							<p>Response:</p>
							<pre>{{ JSON.stringify(evopt.res, null, 2) }}</pre>
						</div>
					</details>
				</div>
				<div v-else>
					<p>nothing to see here</p>
				</div>
			</main>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import Header from "../components/Top/Header.vue";
import store from "../store";
import formatter from "../mixins/formatter";

export default defineComponent({
	name: "Optimize",
	components: {
		TopHeader: Header,
	},
	mixins: [formatter],
	head() {
		return { title: this.$t("energy.title") };
	},
	computed: {
		evopt() {
			return store.state.evopt;
		},
		timeSlots() {
			// Use the dt array length to determine number of time slots
			return this.evopt?.req.time_series.dt || [];
		},
		currency() {
			return store.state.currency;
		},
		gridFeedinPriceLabel() {
			return `Grid Feedin (${this.pricePerKWhUnit(this.currency)})`;
		},
		gridImportPriceLabel() {
			return `Grid Import (${this.pricePerKWhUnit(this.currency)})`;
		},
		statusBadgeClass() {
			if (!this.evopt?.res.status) return "bg-secondary";

			switch (this.evopt.res.status) {
				case "Optimal":
					return "bg-success";
				case "Infeasible":
					return "bg-danger";
				case "Unbounded":
					return "bg-warning";
				default:
					return "bg-secondary";
			}
		},
	},
	methods: {
		formatPower(watts: number): string {
			return (watts / 1000).toFixed(1);
		},
		formatEnergy(wh: number): string {
			return (wh / 1000).toFixed(1);
		},
		formatPowerRange(min: number, max: number): string {
			return `${this.formatPower(min)} - ${this.formatPower(max)}`;
		},
		formatEnergyRange(min: number, max: number): string {
			return `${this.formatEnergy(min)} - ${this.formatEnergy(max)}`;
		},

		formatDuration(seconds: number): string {
			return (seconds / 3600).toFixed(1);
		},
		formatHour(index: number): string {
			const hour = index % 24;
			return hour.toString();
		},
	},
});
</script>

<style scoped>
.table td,
.table th {
	font-variant-numeric: tabular-nums;
}

.table td:not(:first-child) {
	padding-left: 1rem;
}
</style>
