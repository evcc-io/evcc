<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader title="Optimize Debug" />
		<div class="alert alert-light mb-5">
			This page is for development purposes only. Gives insights into the upcoming
			optimization algorithm.
		</div>
		<div class="row">
			<main class="col-12">
				<div v-if="evopt">
					<!-- Optimizer Plan -->
					<section class="mb-5">
						<h3
							class="fw-normal d-flex gap-3 flex-wrap d-flex align-items-baseline overflow-hidden mb-4"
						>
							<span class="d-block no-wrap text-truncate">Result: Charging Plan</span>
							<small class="d-block no-wrap text-truncate">
								{{ evopt.res.status }} ・
								{{
									fmtMoney(
										(evopt.res.objective_value || 0) * -1,
										currency,
										true,
										true
									)
								}}
								saved
							</small>
						</h3>
						<ChargeChart
							:evopt="evopt"
							:battery-details="evopt.details.batteryDetails"
							:timestamp="evopt.details.timestamp[0]"
							:currency="currency"
							:battery-colors="batteryColors"
						/>

						<h3 class="fw-normal mb-4">Result: SoC Projection</h3>
						<SocChart
							:evopt="evopt"
							:battery-details="evopt.details.batteryDetails"
							:timestamp="evopt.details.timestamp[0]"
							:currency="currency"
							:battery-colors="batteryColors"
						/>
					</section>

					<!-- Input Parameters -->
					<section class="mb-5">
						<h3 class="fw-normal mb-4">Input: Grid Prices</h3>
						<PriceChart
							:evopt="evopt"
							:timestamp="evopt.details.timestamp[0]"
							:currency="currency"
						/>

						<h3
							class="fw-normal d-flex gap-3 flex-wrap d-flex align-items-baseline overflow-hidden mb-4"
						>
							<span class="d-block no-wrap text-truncate"> Input: Battery </span>
							<small class="d-block no-wrap text-truncate">
								{{ fmtPercentage((evopt.req.eta_c || 1) * 100, 1) }} charge
								efficiency ・
								{{ fmtPercentage((evopt.req.eta_d || 1) * 100, 1) }} discharge
								efficiency
							</small>
						</h3>

						<BatteryConfigurationTable
							:batteries="evopt.req.batteries"
							:battery-details="evopt.details.batteryDetails"
							:currency="currency"
						/>
					</section>

					<hr class="my-5" />

					<!-- Debugging -->
					<section class="mb-5">
						<h3 class="fw-normal mb-4">Time Series</h3>

						<TimeSeriesDataTable
							:evopt="evopt"
							:battery-details="evopt.details.batteryDetails"
							:timestamp="evopt.details.timestamp[0]"
							:currency="currency"
							:battery-colors="batteryColors"
							:dimmed-battery-colors="dimmedBatteryColors"
						/>

						<h3 class="fw-normal mb-4">Raw Data</h3>

						<div class="mb-4">
							<p class="mb-2">Request:</p>
							<div class="position-relative">
								<pre
									class="p-3 rounded border overflow-auto"
									style="background-color: var(--evcc-box)"
									>{{ formattedRequest }}</pre
								>
								<CopyButton :content="formattedRequest" />
							</div>
						</div>

						<div class="mb-4">
							<p class="mb-2">Response:</p>
							<div class="position-relative">
								<pre
									class="p-3 rounded border overflow-auto"
									style="background-color: var(--evcc-box)"
									>{{ formattedResponse }}</pre
								>
								<CopyButton :content="formattedResponse" />
							</div>
						</div>
					</section>
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
import CopyButton from "../components/Optimize/CopyButton.vue";
import { formatCompactJson } from "../components/Optimize/compactJson";
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
		CopyButton,
	},
	mixins: [formatter],
	head() {
		return { title: "Optimize Debug" };
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
				(_, index) => colors.palette[index % colors.palette.length] || ""
			);
		},
		dimmedBatteryColors() {
			return (this.batteryColors || []).map((color) => this.dimColorBy25Percent(color));
		},
		formattedRequest() {
			return this.evopt?.req ? formatCompactJson(this.evopt.req) : "";
		},
		formattedResponse() {
			return this.evopt?.res ? formatCompactJson(this.evopt.res) : "";
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
