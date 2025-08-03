<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader title="Optimize" />
		<div class="alert alert-light mb-4">This page is work in progress.</div>
		<div class="row">
			<main class="col-12">
				<div v-if="evopt">
					<div class="mb-4">
						<h3 class="fw-normal mb-3">Battery Configuration</h3>
						<div class="table-responsive">
							<table class="table">
								<thead>
									<tr>
										<th scope="col">#</th>
										<th scope="col">Max Charge (W)</th>
										<th scope="col">Min Charge (W)</th>
										<th scope="col">Max Discharge (W)</th>
										<th scope="col">Aux Power</th>
										<th scope="col">Initial SoC (Wh)</th>
										<th scope="col">Max SoC (Wh)</th>
										<th scope="col">Min SoC (Wh)</th>
									</tr>
								</thead>
								<tbody>
									<tr
										v-for="(battery, index) in evopt.req.batteries"
										:key="index"
									>
										<th scope="row">{{ index + 1 }}</th>
										<td>{{ formatPower(battery.c_max) }}</td>
										<td>{{ formatPower(battery.c_min) }}</td>
										<td>{{ formatPower(battery.d_max) }}</td>
										<td>{{ battery.p_a }}</td>
										<td>{{ formatEnergy(battery.s_initial) }}</td>
										<td>{{ formatEnergy(battery.s_max) }}</td>
										<td>{{ formatEnergy(battery.s_min) }}</td>
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
											Flow Direction
										</td>
										<td
											v-for="(value, index) in evopt.res.flow_direction"
											:key="index"
											:class="['text-end', { 'text-muted': value === 0 }]"
										>
											{{ value }}
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
	},
	methods: {
		formatPower(watts: number): string {
			return (watts / 1000).toFixed(1);
		},
		formatEnergy(wh: number): string {
			return (wh / 1000).toFixed(1);
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
