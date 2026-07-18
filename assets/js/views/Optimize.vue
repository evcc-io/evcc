<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader title="Optimize Debug 🧪" />
		<Card edge-to-edge class="box-pull-out mt-4 mb-4">
			<OptimizeHeader
				:updated="evopt?.updated"
				:status="evopt?.res?.status"
				:net-cost="netCost"
				:horizon-hours="horizonHours"
				:currency="currency"
				:charging-strategies="chargingStrategies"
				:selected-strategy="optimizerChargingStrategy"
				:pending="pending"
				@optimize="optimizeNow"
				@change-strategy="changeChargingStrategy"
			/>
		</Card>
		<div class="row">
			<main class="col-12">
				<div v-if="evopt">
					<Card title="Result: Charging Plan" edge-to-edge class="box-pull-out mb-4">
						<ChargeChart
							:evopt="evopt"
							:battery-details="evopt.details.batteryDetails"
							:timestamp="evopt.details.timestamp[0]"
							:currency="currency"
							:battery-colors="batteryColors"
							:device-colors="deviceColors"
						/>
					</Card>

					<Card title="Result: SoC Projection" edge-to-edge class="box-pull-out mb-4">
						<SocChart
							:evopt="evopt"
							:battery-details="evopt.details.batteryDetails"
							:timestamp="evopt.details.timestamp[0]"
							:currency="currency"
							:battery-colors="batteryColors"
						/>
					</Card>

					<Card title="Input: Grid Prices" edge-to-edge class="box-pull-out mb-4">
						<PriceChart
							:evopt="evopt"
							:timestamp="evopt.details.timestamp[0]"
							:currency="currency"
						/>
					</Card>

					<Card
						title="Input: Battery"
						:subtitle="batteryEfficiencySubtitle"
						edge-to-edge
						class="box-pull-out mb-4"
					>
						<BatteryConfigurationTable
							:batteries="evopt.req.batteries"
							:battery-details="evopt.details.batteryDetails"
							:currency="currency"
						/>
					</Card>

					<Card title="Time Series" edge-to-edge class="box-pull-out mb-4">
						<TimeSeriesDataTable
							:evopt="evopt"
							:battery-details="evopt.details.batteryDetails"
							:timestamps="evopt.details.timestamp"
							:currency="currency"
							:battery-colors="batteryColors"
							:dimmed-battery-colors="dimmedBatteryColors"
						/>
					</Card>

					<Card title="Raw Data" edge-to-edge class="box-pull-out mb-4">
						<div class="mb-4">
							<p class="mb-2">Request:</p>
							<div class="position-relative">
								<pre
									class="p-3 overflow-auto"
									style="background-color: var(--evcc-gray-10)"
									>{{ formattedRequest }}</pre
								>
								<CopyButton :content="formattedRequest" />
							</div>
						</div>

						<div>
							<p class="mb-2">Response:</p>
							<div class="position-relative">
								<pre
									class="p-3 overflow-auto"
									style="background-color: var(--evcc-gray-10)"
									>{{ formattedResponse }}</pre
								>
								<CopyButton :content="formattedResponse" />
							</div>
						</div>
					</Card>
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
import Card from "../components/Helper/Card.vue";
import OptimizeHeader from "../components/Optimize/OptimizeHeader.vue";
import BatteryConfigurationTable from "../components/Optimize/BatteryConfigurationTable.vue";
import SocChart from "../components/Optimize/SocChart.vue";
import ChargeChart from "../components/Optimize/ChargeChart.vue";
import PriceChart from "../components/Optimize/PriceChart.vue";
import TimeSeriesDataTable from "../components/Optimize/TimeSeriesDataTable.vue";
import CopyButton from "../components/Optimize/CopyButton.vue";
import { formatCompactJson } from "../components/Optimize/compactJson";
import api from "../api";
import store from "../store";
import formatter from "../mixins/formatter";
import { resolveColors, deviceColorMap } from "../colors";
import { CURRENCY } from "../types/evcc";

export default defineComponent({
	name: "Optimize",
	components: {
		TopHeader: Header,
		Card,
		OptimizeHeader,
		BatteryConfigurationTable,
		SocChart,
		ChargeChart,
		PriceChart,
		TimeSeriesDataTable,
		CopyButton,
	},
	mixins: [formatter],
	data() {
		return {
			pending: false,
		};
	},
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
		chargingStrategies(): string[] {
			return store.state.optimizerChargingStrategies || [];
		},
		optimizerChargingStrategy(): string {
			return store.state.optimizerChargingStrategy || "";
		},
		netCost(): number {
			return (this.evopt?.res?.objective_value || 0) * -1;
		},
		horizonHours(): number {
			const dt = this.evopt?.req?.time_series?.dt;
			if (!dt?.length) return 0;
			return Math.round(dt.reduce((sum, s) => sum + s, 0) / 3600);
		},
		batteryEfficiencySubtitle(): string {
			const etaC = this.fmtPercentage((this.evopt?.req.eta_c || 1) * 100, 1);
			const etaD = this.fmtPercentage((this.evopt?.req.eta_d || 1) * 100, 1);
			return `${etaC} charge efficiency ・ ${etaD} discharge efficiency`;
		},
		deviceColors() {
			return deviceColorMap(store.state.deviceColors);
		},
		batteryTitles(): string[] {
			const details = this.evopt?.details?.batteryDetails || [];
			return (this.evopt?.res.batteries || []).map(
				(_, i) => details[i]?.title || details[i]?.name || `battery-${i}`
			);
		},
		batteryColors() {
			if (!this.evopt?.res.batteries) return [];
			const palette = resolveColors(this.batteryTitles, this.deviceColors);
			return this.batteryTitles.map((t) => palette[t] || "");
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
	watch: {
		"evopt.updated"() {
			// re-enable the refresh action once a fresh optimizer run lands
			this.pending = false;
		},
	},
	methods: {
		optimizeNow() {
			this.pending = true;
			api.post("optimize");
		},
		changeChargingStrategy(value: string) {
			api.post(`optimizerchargingstrategy/${value}`);
		},
		dimColorBy25Percent(color: string): string {
			// Convert color to 25% opacity (40 in hex = 25% of 255)
			return color?.toLowerCase().replace(/ff$/, "40") || color;
		},
	},
});
</script>
