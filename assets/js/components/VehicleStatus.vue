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
			if (!this.enabled && this.pvAction === "enable") {
				return t("pvEnable", {
					remaining: this.fmtShortDuration(this.pvRemainingInterpolated, true),
				});
			}

			// waiting for vehicle
			if (this.enabled && !this.charging) {
				return t("waitForVehicle");
			}

			// pv enable
			if (this.charging && this.pvAction === "disable") {
				return t("pvDisable", {
					remaining: this.fmtShortDuration(this.pvRemainingInterpolated, true),
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
