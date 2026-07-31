<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader :title="$t('batterySettings.modalTitle')" />
		<div class="row">
			<main class="col-12">
				<BatteryExperimental v-if="experimental" />
				<template v-else-if="batteryAvailable">
					<h3 class="fw-normal my-4">
						{{ $t("batterySettings.usageTab") }}
					</h3>
					<BatteryUsageSettings
						:buffer-soc="state.bufferSoc"
						:priority-soc="state.prioritySoc"
						:buffer-start-soc="state.bufferStartSoc"
						:battery-discharge-control="state.batteryDischargeControl"
						:battery="state.battery"
					/>
					<template v-if="gridChargeVisible">
						<hr class="my-5" />
						<h3 class="fw-normal my-4 mt-5">
							{{ $t("batterySettings.gridChargeTab") }}
						</h3>
						<SmartCostLimit v-bind="smartCostLimitProps" />
					</template>
				</template>
				<p v-else class="my-4 text-muted">
					{{ $t("batterySettings.noBattery") }}
				</p>
			</main>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import Header from "../components/Top/Header.vue";
import BatteryUsageSettings from "../components/Battery/BatteryUsageSettings.vue";
import BatteryExperimental from "../components/Battery/BatteryExperimental.vue";
import SmartCostLimit from "../components/Tariff/SmartCostLimit.vue";
import store from "../store";
import settings from "../settings";
import { SMART_COST_TYPE } from "../types/evcc";

export default defineComponent({
	name: "Battery",
	components: {
		TopHeader: Header,
		BatteryUsageSettings,
		BatteryExperimental,
		SmartCostLimit,
	},
	head() {
		return { title: this.$t("batterySettings.modalTitle") };
	},
	computed: {
		experimental(): boolean {
			return !!store.state.experimental;
		},
		state() {
			return store.state;
		},
		batteryAvailable() {
			return (this.state.battery?.devices?.length ?? 0) > 0;
		},
		gridChargePossible() {
			const devices = this.state.battery?.devices ?? [];
			return (
				devices.some(({ controllable }) => controllable) && this.state.smartCostAvailable
			);
		},
		gridChargeLimit() {
			return this.state.batteryGridChargeLimit ?? null;
		},
		gridChargeVisible() {
			return this.gridChargePossible || this.gridChargeLimit !== null;
		},
		gridChargeTariff() {
			const { forecast, smartCostType } = this.state;
			return smartCostType === SMART_COST_TYPE.CO2 ? forecast?.co2 : forecast?.grid;
		},
		smartCostLimitProps() {
			return {
				currentLimit: this.gridChargeLimit,
				lastLimit: settings.lastBatterySmartCostLimit,
				smartCostType: this.state.smartCostType,
				currency: this.state.currency,
				tariff: this.gridChargeTariff,
				possible: this.gridChargePossible,
			};
		},
	},
});
</script>
