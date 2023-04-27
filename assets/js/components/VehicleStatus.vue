<template>
	<div class="d-block evcc-gray">{{ message }}&nbsp;</div>
</template>

<script>
import formatter from "../mixins/formatter";
import { CO2_UNIT } from "../units";

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
		targetTime: String,
		planProjectedStart: String,
		phaseAction: String,
		phaseRemainingInterpolated: Number,
		pvAction: String,
		pvRemainingInterpolated: Number,
		guardAction: String,
		guardRemainingInterpolated: Number,
		targetChargeDisabled: Boolean,
		climaterActive: Boolean,
		smartCostLimit: Number,
		smartCostUnit: String,
		tariffGrid: Number,
		tariffCo2: Number,
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
		isCo2() {
			return this.smartCostUnit === CO2_UNIT;
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

			// target charge
			if (this.targetTime && !this.targetChargeDisabled) {
				if (this.charging) {
					return t("targetChargeActive");
				}
				if (this.enabled) {
					return t("targetChargeWaitForVehicle");
				}
				if (this.planProjectedStart) {
					return t("targetChargePlanned", {
						time: this.fmtAbsoluteDate(new Date(this.planProjectedStart)),
					});
				}
			}

			// clean energy
			if (this.charging && this.isCo2 && this.tariffCo2 < this.smartCostLimit) {
				return t("cleanEnergyCharging");
			}

			// cheap energy
			if (this.charging && !this.isCo2 && this.tariffGrid < this.smartCostLimit) {
				return t("cheapEnergyCharging");
			}

			if (this.pvTimerActive && !this.enabled && this.pvAction === "enable") {
				return t("pvEnable", {
					remaining: this.fmtShortDuration(this.pvRemainingInterpolated, true),
				});
			}

			if (this.enabled && this.climaterActive) {
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
					remaining: this.fmtShortDuration(this.pvRemainingInterpolated, true),
				});
			}

			if (this.phaseTimerActive) {
				return t(this.phaseAction, {
					remaining: this.fmtShortDuration(this.phaseRemainingInterpolated, true),
				});
			}

			if (this.charging) {
				return t("charging");
			}

			if (this.guardTimerActive) {
				return t("guard", {
					remaining: this.fmtShortDuration(this.guardRemainingInterpolated, true),
				});
			}

			return t("connected");
		},
	},
};
</script>
