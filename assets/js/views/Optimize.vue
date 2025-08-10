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

					<BatteryConfigurationTable
						:batteries="evopt.req.batteries"
						:battery-details="evopt.details.batteryDetails"
						:currency="currency"
					/>

					<ChargeChart
						:evopt="evopt"
						:battery-details="evopt.details.batteryDetails"
						:timestamp="evopt.details.timestamp[0]"
						:currency="currency"
						:battery-colors="batteryColors"
					/>
					<SocChart
						:evopt="evopt"
						:battery-details="evopt.details.batteryDetails"
						:timestamp="evopt.details.timestamp[0]"
						:currency="currency"
						:battery-colors="batteryColors"
					/>
					<PriceChart
						:evopt="evopt"
						:timestamp="evopt.details.timestamp[0]"
						:currency="currency"
					/>

					<TimeSeriesDataTable
						:evopt="evopt"
						:battery-details="evopt.details.batteryDetails"
						:timestamp="evopt.details.timestamp[0]"
						:currency="currency"
						:battery-colors="batteryColors"
						:dimmed-battery-colors="dimmedBatteryColors"
					/>

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
import BatteryConfigurationTable from "../components/Optimize/BatteryConfigurationTable.vue";
import SocChart from "../components/Optimize/SocChart.vue";
import ChargeChart from "../components/Optimize/ChargeChart.vue";
import PriceChart from "../components/Optimize/PriceChart.vue";
import TimeSeriesDataTable from "../components/Optimize/TimeSeriesDataTable.vue";
import store from "../store";
import formatter from "../mixins/formatter";
import colors from "../colors";
import { CURRENCY } from "../types/evcc";

export default defineComponent({
	name: "Optimize",
	components: {
		TopHeader: Header,
		BatteryConfigurationTable,
		SocChart,
		ChargeChart,
		PriceChart,
		TimeSeriesDataTable,
	},
	mixins: [formatter],
	head() {
		return { title: this.$t("energy.title") };
	},
	computed: {
		evopt() {
			return store.state.evopt;
		},
		currency() {
			return store.state.currency || CURRENCY.EUR;
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
		batteryColors() {
			if (!this.evopt?.res.batteries) return [];

			return this.evopt.res.batteries.map(
				(_, index) => colors.palette[index % colors.palette.length]
			);
		},
		dimmedBatteryColors() {
			return this.batteryColors.map((color) => this.dimColorBy25Percent(color));
		},
	},
	methods: {
		dimColorBy25Percent(color: string): string {
			// Convert color to 25% opacity (40 in hex = 25% of 255)
			return color?.toLowerCase().replace(/ff$/, "40") || color;
		},
	},
});
</script>
