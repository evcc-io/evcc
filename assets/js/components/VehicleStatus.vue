<template>
	<div class="d-block text-gray">{{ message }}&nbsp;</div>
</template>

<script>
import collector from "../mixins/collector";
import formatter from "../mixins/formatter";

export default {
	name: "VehicleStatus",
	mixins: [collector, formatter],
	props: {
		vehicleSoC: Number,
		minSoC: Number,
		enabled: Boolean,
		connected: Boolean,
		charging: Boolean,
		targetTime: String,
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

			// min charge active
			if (this.minSoC > 0 && this.vehicleSoC < this.minSoC) {
				return t("minCharge", { soc: this.minSoC });
			}

			// target charage
			if (this.targetTime) {
				const targetDate = new Date(this.targetTime);
				const data = { time: this.fmtAbsoluteDate(targetDate) };

				if (this.charging) {
					return t("targetChargeActive", data);
				}
				if (this.enabled) {
					return t("targetChargeWaitForVehicle", data);
				}
				return t("targetChargePlanned", data);
			}

			// pv enable
			if (this.pvTimerActive && !this.enabled && this.pvAction === "enable") {
				return t("pvEnable", {
					remaining: this.fmtShortDuration(this.pvRemainingInterpolated, true),
				});
			}

			// waiting for vehicle
			if (this.enabled && !this.charging) {
				return t("waitForVehicle");
			}

			// pv disable
			if (this.pvTimerActive && this.charging && this.pvAction === "disable") {
				return t("pvDisable", {
					remaining: this.fmtShortDuration(this.pvRemainingInterpolated, true),
				});
			}

			// phase timer
			if (this.phaseTimerActive) {
				return t(this.phaseAction, {
					remaining: this.fmtShortDuration(this.phaseRemainingInterpolated, true),
				});
			}

			// charging
			if (this.charging) {
				return t("charging");
			}

			// connected
			if (this.connected) {
				return t("connected");
			}

			return "";
		},
	},
};
</script>
