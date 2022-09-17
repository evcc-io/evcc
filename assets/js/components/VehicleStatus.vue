<template>
	<div class="d-block evcc-gray">{{ message }}&nbsp;</div>
</template>

<script>
import formatter from "../mixins/formatter";

export default {
	name: "VehicleStatus",
	mixins: [formatter],
	props: {
		vehicleSoC: Number,
		vehicleTargetSoC: Number,
		minSoC: Number,
		enabled: Boolean,
		connected: Boolean,
		charging: Boolean,
		targetTime: String,
		targetTimeProjectedStart: String,
		phaseAction: String,
		phaseRemainingInterpolated: Number,
		pvAction: String,
		pvRemainingInterpolated: Number,
	},
	computed: {
		phaseTimerActive() {
			return (
				this.charging &&
				this.phaseRemainingInterpolated > 0 &&
				["scale1p", "scale3p"].includes(this.phaseAction)
			);
		},
		pvTimerActive() {
			return (
				this.pvRemainingInterpolated > 0 && ["enable", "disable"].includes(this.pvAction)
			);
		},
		message: function () {
			const t = (key, data) => {
				return this.$t(`main.vehicleStatus.${key}`, data);
			};

			if (!this.connected) {
				return t("disconnected");
			}

			// min charge active
			if (this.minSoC > 0 && this.vehicleSoC < this.minSoC) {
				return t("minCharge", { soc: this.minSoC });
			}

			// target charage
			if (this.targetTime) {
				if (this.charging) {
					return t("targetChargeActive");
				}
				if (this.enabled) {
					return t("targetChargeWaitForVehicle");
				}
				if (this.targetTimeProjectedStart) {
					return t("targetChargePlanned", {
						time: this.fmtAbsoluteDate(new Date(this.targetTimeProjectedStart)),
					});
				}
			}

			if (this.pvTimerActive && !this.enabled && this.pvAction === "enable") {
				return t("pvEnable", {
					remaining: this.fmtShortDuration(this.pvRemainingInterpolated, true),
				});
			}

			if (this.enabled && !this.charging) {
				if (this.vehicleTargetSoC > 0 && this.vehicleSoC >= this.vehicleTargetSoC - 1) {
					return t("vehicleTargetReached", { soc: this.vehicleTargetSoC });
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

			return t("connected");
		},
	},
};
</script>
