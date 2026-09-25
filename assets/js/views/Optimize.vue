<template>
	<div
		class="container px-4 safe-area-inset d-flex flex-column"
		:class="{ 'empty-container': !evopt }"
	>
		<TopHeader title="Optimize Debug 🧪" />
		<div v-if="!evopt" class="flex-grow-1 d-flex" data-testid="optimize-empty">
			<div class="empty-box d-flex flex-column p-5">
				<p class="text-muted">
					The optimizer is enabled and collecting data. For new installations this can
					take up to 24 hours.
				</p>
				<p class="text-muted mb-4">
					If nothing shows up after that, the logs may tell why.
				</p>
				<router-link
					:to="{ path: '/log', query: { q: 'optimizer' } }"
					class="btn btn-outline-primary"
				>
					Check logs
				</router-link>
			</div>
		</div>
		<Card v-else edge-to-edge class="box-pull-out mt-4 mb-4">
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
			<AutomaticModeStrip
				:automatic="optimizerAutomatic"
				:is-sponsor="isSponsor"
				@change="changeAutomatic"
				@learn-more="openOptimizerModal"
			/>
		</Card>
		<div v-if="evopt" class="row">
			<main class="col-12">
				<div>
					<h2 class="mt-2 mb-4">Optimizer Plan</h2>

					<Card
						title="Charging Plan"
						subtitle="kW"
						edge-to-edge
						class="box-pull-out mb-4"
					>
						<ChargeChart
							:evopt="evopt"
							:battery-details="evopt.details.batteryDetails"
							:demand-details="demandDetails"
							:demand-colors="demandColors"
							:timestamp="evopt.details.timestamp[0]"
							:battery-colors="batteryColors"
							:device-colors="deviceColors"
						/>
					</Card>

					<Card
						v-if="socEntries.length"
						title="SoC Projections"
						subtitle="%"
						edge-to-edge
						class="box-pull-out mb-4"
					>
						<div
							v-for="(entry, idx) in socEntries"
							:key="entry.index"
							:class="{ 'mb-3': idx < socEntries.length - 1 }"
						>
							<SocChart
								:evopt="evopt"
								:entry="entry"
								:timestamp="evopt.details.timestamp[0]"
								:show-x-axis="idx === socEntries.length - 1"
							/>
						</div>
					</Card>

					<Card
						title="Timeline"
						:subtitle="timeSeriesSubtitle"
						edge-to-edge
						class="box-pull-out mb-4"
					>
						<TimeSeriesDataTable
							:evopt="evopt"
							mode="response"
							:battery-details="evopt.details.batteryDetails"
							:timestamps="evopt.details.timestamp"
							:currency="currency"
							:battery-colors="batteryColors"
						/>
					</Card>

					<Card title="Feedback" edge-to-edge class="box-pull-out mb-4">
						<p>
							Unexpected results or implausible numbers? Open an issue in the
							optimizer repository and attach the request and response below.
						</p>
						<a
							href="https://github.com/evcc-io/optimizer/issues"
							target="_blank"
							class="btn btn-outline-primary"
						>
							Open issue
						</a>
					</Card>

					<h2 class="section-title mb-4">Optimizer Inputs</h2>

					<Card
						title="Batteries"
						:subtitle="batteryEfficiencySubtitle"
						edge-to-edge
						class="box-pull-out mb-4"
					>
						<BatteryConfigurationTable
							:batteries="evopt.req.batteries"
							:battery-details="evopt.details.batteryDetails"
							:battery-colors="batteryColors"
							:currency="currency"
						/>
					</Card>

					<Card
						title="Environment"
						:subtitle="timeSeriesSubtitle"
						edge-to-edge
						class="box-pull-out mb-4"
					>
						<TimeSeriesDataTable
							:evopt="evopt"
							mode="request"
							:battery-details="evopt.details.batteryDetails"
							:demand-details="demandDetails"
							:demand-colors="demandColors"
							:timestamps="evopt.details.timestamp"
							:currency="currency"
							:battery-colors="batteryColors"
						/>
					</Card>

					<h2 class="section-title mb-4">Raw Data</h2>

					<Card title="Request" edge-to-edge class="box-pull-out mb-4">
						<div class="position-relative">
							<pre
								class="p-3 overflow-auto"
								style="background-color: var(--evcc-gray-10)"
								>{{ formattedRequest }}</pre>
							<CopyButton :content="formattedRequest" />
						</div>
					</Card>

					<Card title="Response" edge-to-edge class="box-pull-out mb-4">
						<div class="position-relative">
							<pre
								class="p-3 overflow-auto"
								style="background-color: var(--evcc-gray-10)"
								>{{ formattedResponse }}</pre>
							<CopyButton :content="formattedResponse" />
						</div>
					</Card>
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
import AutomaticModeStrip from "../components/Optimize/AutomaticModeStrip.vue";
import { openModal } from "../configModal";
import BatteryConfigurationTable from "../components/Optimize/BatteryConfigurationTable.vue";
import SocChart, { type SocChartEntry } from "../components/Optimize/SocChart.vue";
import ChargeChart from "../components/Optimize/ChargeChart.vue";
import TimeSeriesDataTable from "../components/Optimize/TimeSeriesDataTable.vue";
import CopyButton from "../components/Optimize/CopyButton.vue";
import { formatCompactJson } from "../components/Optimize/compactJson";
import { loadpointTitle, type Titled } from "../components/Optimize/chart";
import api from "../api";
import store from "../store";
import formatter from "../mixins/formatter";
import { resolveColors, deviceColorMap, batteryColor } from "../colors";
import { CURRENCY, type BatteryDetail, type DemandDetail } from "../types/evcc";

export default defineComponent({
	name: "Optimize",
	components: {
		TopHeader: Header,
		Card,
		OptimizeHeader,
		BatteryConfigurationTable,
		SocChart,
		ChargeChart,
		TimeSeriesDataTable,
		CopyButton,
		AutomaticModeStrip,
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
		optimizerAutomatic(): boolean {
			return !!store.state.optimizerAutomatic;
		},
		isSponsor(): boolean {
			return !!store.state.sponsor?.status?.name;
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
		batteryDetails(): BatteryDetail[] {
			return this.evopt?.details?.batteryDetails || [];
		},
		demandDetails(): DemandDetail[] {
			return this.evopt?.details?.demandDetails || [];
		},
		// vehicle batteries and titled demand profiles are loadpoints
		loadpointColorKeys(): string[] {
			const details: Titled[] = [
				...this.batteryDetails.filter((d) => d.type === "vehicle"),
				...this.demandDetails.filter((d) => d.title),
			];
			return [...new Set(details.map(loadpointTitle))];
		},
		// loadpoints share the picker palette with History
		loadpointPalette() {
			return resolveColors(this.loadpointColorKeys, this.deviceColors);
		},
		// per-entry colors aligned with res.batteries: dedicated battery palette
		// for home batteries (same as battery page and history), picker palette
		// for loadpoints
		batteryColors(): string[] {
			let batteryIndex = 0;
			return this.batteryDetails.map((d) => {
				if (d.type === "battery") return batteryColor(batteryIndex++);
				return this.loadpointColor(d);
			});
		},
		// per-entry colors aligned with demandDetails, untitled rows stay muted
		demandColors(): string[] {
			return this.demandDetails.map(this.loadpointColor);
		},
		// loadpoints first, then batteries, matching the charging plan order
		socEntries(): SocChartEntry[] {
			const entries = this.batteryDetails.map((d, i) => ({
				index: i,
				type: d.type,
				title: d.title || d.name,
				capacity: d.capacity,
				color: this.batteryColors[i] || "",
			}));
			return [
				...entries.filter((e) => e.type === "vehicle"),
				...entries.filter((e) => e.type === "battery"),
			];
		},
		timeSeriesSubtitle(): string {
			return `${this.evopt?.req.time_series.dt.length || 0} steps ・ ${this.horizonHours} h horizon`;
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
		loadpointColor(detail: Titled): string {
			return this.loadpointPalette[loadpointTitle(detail)] || "";
		},
		optimizeNow() {
			this.pending = true;
			api.post("optimize");
		},
		changeChargingStrategy(value: string) {
			api.post(`optimizerchargingstrategy/${value}`);
		},
		changeAutomatic(checked: boolean) {
			api.post(`config/optimizerautomatic/${checked}`);
		},
		openOptimizerModal() {
			openModal("optimizer");
		},
	},
});
</script>

<style scoped>
.section-title {
	margin-top: 4rem;
}
</style>
