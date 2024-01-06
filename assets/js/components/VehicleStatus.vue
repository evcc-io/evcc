<template>
	<div class="d-block evcc-gray" data-testid="vehicle-status">{{ message }}&nbsp;</div>
</template>

<script>
import formatter from "../mixins/formatter";
import { CO2_TYPE } from "../units";

export default {
	name: "VehicleStatus",
	mixins: [formatter],
	props: {
		vehicleSoc: Number,
		vehicleTargetSoc: Number,
		minSoc: Number,
		enabled: Boolean,
		connected: Boolean,
		charging: Boolean,
		effectivePlanTime: String,
		planProjectedStart: String,
		planActive: Boolean,
		phaseAction: String,
		phaseRemainingInterpolated: Number,
		pvAction: String,
		pvRemainingInterpolated: Number,
		guardAction: String,
		guardRemainingInterpolated: Number,
		targetChargeDisabled: Boolean,
		vehicleClimaterActive: Boolean,
		smartCostLimit: Number,
		smartCostType: String,
		smartCostActive: Boolean,
		tariffGrid: Number,
		tariffCo2: Number,
		currency: String,
	},
	computed: {
		phaseTimerActive() {
			return (
				this.phaseRemainingInterpolated > 0 &&
				["scale1p", "scale3p"].includes(this.phaseAction)
			);
		},
		pvTimerActive() {
			return (
				this.pvRemainingInterpolated > 0 && ["enable", "disable"].includes(this.pvAction)
			);
		},
		guardTimerActive() {
			return this.guardRemainingInterpolated > 0 && this.guardAction === "enable";
		},
		message: function () {
			const t = (key, data) => {
				return this.$t(`main.vehicleStatus.${key}`, data);
			};

			if (!this.connected) {
				return t("disconnected");
			}
			// min charge active
			if (this.minSoc > 0 && this.vehicleSoc < this.minSoc) {
				return t("minCharge", { soc: this.minSoc });
			}

			// plan
			if (this.effectivePlanTime && !this.targetChargeDisabled) {
				if (this.planActive && this.charging) {
					return t("targetChargeActive");
				}
				if (this.planActive && this.enabled) {
					return t("targetChargeWaitForVehicle");
				}
				if (this.planProjectedStart) {
					return t("targetChargePlanned", {
						time: this.fmtAbsoluteDate(new Date(this.planProjectedStart)),
					});
				}
			}

			// clean or cheap energy
			if (this.charging && this.smartCostActive) {
				return this.smartCostType === CO2_TYPE
					? t("cleanEnergyCharging", {
							co2: this.fmtCo2Short(this.tariffCo2),
							limit: this.fmtCo2Short(this.smartCostLimit),
						})
					: t("cheapEnergyCharging", {
							price: this.fmtPricePerKWh(this.tariffGrid, this.currency, true),
							limit: this.fmtPricePerKWh(this.smartCostLimit, this.currency, true),
						});
			}

			if (this.pvTimerActive && !this.enabled && this.pvAction === "enable") {
				return t("pvEnable", {
					remaining: this.fmtDuration(this.pvRemainingInterpolated),
				});
			}

			if (this.enabled && this.vehicleClimaterActive) {
				return t("climating");
			}

			if (this.enabled && !this.charging) {
				if (this.vehicleTargetSoc > 0 && this.vehicleSoc >= this.vehicleTargetSoc - 1) {
					return t("vehicleTargetReached", { soc: this.vehicleTargetSoc });
				}
				return t("waitForVehicle");
			}

			if (this.pvTimerActive && this.charging && this.pvAction === "disable") {
				return t("pvDisable", {
					remaining: this.fmtDuration(this.pvRemainingInterpolated),
				});
			}

			if (this.phaseTimerActive) {
				return t(this.phaseAction, {
					remaining: this.fmtDuration(this.phaseRemainingInterpolated),
				});
			}

			if (this.guardTimerActive) {
				return t("guard", {
					remaining: this.fmtDuration(this.guardRemainingInterpolated),
				});
			}

			if (this.charging) {
				return t("charging");
			}

			return t("connected");
		},
	},
};
</script>
