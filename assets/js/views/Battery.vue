<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader :title="$t('batterySettings.modalTitle')" />
		<div class="row">
			<main class="col-12">
				<template v-if="batteryAvailable">
					<h3 class="fw-normal my-4">
						{{ $t("batterySettings.usageTab") }}
					</h3>
					<BatteryUsageSettings style="max-width: 950px" v-bind="batteryUsageProps" />
					<template v-if="gridChargePossible">
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
import SmartCostLimit from "../components/Tariff/SmartCostLimit.vue";
import store from "../store";
import settings from "../settings";
import collector from "../mixins/collector";
import { SMART_COST_TYPE } from "../types/evcc";

export default defineComponent({
	name: "Battery",
	components: {
		TopHeader: Header,
		BatteryUsageSettings,
		SmartCostLimit,
	},
	mixins: [collector],
	head() {
		return { title: this.$t("batterySettings.modalTitle") };
	},
	computed: {
		state() {
			return store.state;
		},
		batteryAvailable() {
			return (this.state.battery?.devices?.length ?? 0) > 0;
		},
		batteryUsageProps() {
			return this.collectProps(BatteryUsageSettings, this.state);
		},
		gridChargePossible() {
			const devices = this.state.battery?.devices ?? [];
			return (
				devices.some(({ controllable }) => controllable) && this.state.smartCostAvailable
			);
		},
		gridChargeTariff() {
			const { forecast, smartCostType } = this.state;
			return smartCostType === SMART_COST_TYPE.CO2 ? forecast?.co2 : forecast?.grid;
		},
		smartCostLimitProps() {
			return {
				currentLimit: this.state.batteryGridChargeLimit ?? null,
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
