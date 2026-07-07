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
					<template v-if="solarControlPossible">
						<hr class="my-5" />
						<h3 class="fw-normal my-4 mt-5">
							{{ $t("batterySettings.batteryControlTab") }}
						</h3>
						<div class="form-check form-switch">
							<input
								id="batterySolarControl"
								:checked="state.batterySolarControl"
								class="form-check-input"
								type="checkbox"
								role="switch"
								@change="changeSolarControl"
							/>
							<label class="form-check-label" for="batterySolarControl">
								{{ $t("batterySettings.batteryControl") }}
							</label>
						</div>
						<template v-if="state.batterySolarControl">
							<div class="mt-3 mb-1 small text-muted fw-semibold text-uppercase">
								{{ $t("batterySettings.batteryControlModeTab") }}
							</div>
							<div class="d-flex gap-4">
								<div class="form-check">
									<input
										id="batterySolarModePerBattery"
										:checked="!state.batterySolarPool"
										class="form-check-input"
										type="radio"
										name="batterySolarMode"
										@change="changePool(false)"
									/>
									<label class="form-check-label" for="batterySolarModePerBattery">
										{{ $t("batterySettings.batteryControlModePerBattery") }}
									</label>
								</div>
								<div class="form-check">
									<input
										id="batterySolarModePool"
										:checked="state.batterySolarPool"
										class="form-check-input"
										type="radio"
										name="batterySolarMode"
										@change="changePool(true)"
									/>
									<label class="form-check-label" for="batterySolarModePool">
										{{ $t("batterySettings.batteryControlModePool") }}
									</label>
								</div>
							</div>
							<p class="text-muted small mt-1">
								{{ state.batterySolarPool ? $t("batterySettings.batteryControlModePoolDesc") : $t("batterySettings.batteryControlModePerBatteryDesc") }}
							</p>
						</template>
						<hr class="my-4" />
						<h3 class="fw-normal my-4">
							{{ $t("batterySettings.calibrationTab") }}
						</h3>
						<div class="form-check form-switch">
							<input
								id="batteryCalibrationCharge"
								:checked="state.batteryCalibrationCharge"
								class="form-check-input"
								type="checkbox"
								role="switch"
								@change="changeCalibrationCharge"
							/>
							<label class="form-check-label" for="batteryCalibrationCharge">
								{{ $t("batterySettings.calibrationLabel") }}
							</label>
						</div>
					</template>
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
import api from "../api";
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
		solarControlPossible() {
			const devices = this.state.battery?.devices ?? [];
			return devices.some(({ controllable }) => controllable);
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
	methods: {
		async changeSolarControl(e: Event) {
			try {
				await api.post(
					`batterysolarcontrol/${(e.target as HTMLInputElement).checked ? "true" : "false"}`
				);
			} catch (err) {
				console.error(err);
			}
		},
		async changeCalibrationCharge(e: Event) {
			try {
				await api.post(
					`batterycalibrationcharge/${(e.target as HTMLInputElement).checked ? "true" : "false"}`
				);
			} catch (err) {
				console.error(err);
			}
		},
		async changePool(value: boolean) {
			try {
				await api.post(`batterysolarpool/${value ? "true" : "false"}`);
			} catch (err) {
				console.error(err);
			}
		},
	},
});
</script>
