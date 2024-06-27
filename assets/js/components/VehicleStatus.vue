<template>
	<div class="d-flex justify-content-between gap-4 evcc-gray" data-testid="vehicle-status">
		<div class="text-nowrap" data-testid="vehicle-status-charger">{{ chargerStatus }}</div>
		<div class="d-flex flex-wrap justify-content-end gap-3">
			<!-- pv/phase timer -->
			<div
				v-if="pvTimerVisible"
				class="entry"
				:class="pvTimerClass"
				data-testid="vehicle-status-pvtimer"
			>
				<SunUpIcon v-if="pvAction === 'enable'" />
				<SunDownIcon v-else />
				<div class="tabular">{{ fmtDuration(pvRemainingInterpolated) }}</div>
			</div>
			<div
				v-else-if="phaseTimerVisible"
				class="entry gap-0 text-primary"
				data-testid="vehicle-status-phasetimer"
			>
				<shopicon-regular-sun></shopicon-regular-sun>
				<shopicon-regular-angledoublerightsmall
					:class="phaseIconClass"
					class="me-1"
				></shopicon-regular-angledoublerightsmall>
				<div class="tabular">{{ fmtDuration(phaseRemainingInterpolated) }}</div>
			</div>
			<div
				v-else-if="solarPercentageVisible"
				class="entry"
				:class="solarPercentageClass"
				data-testid="vehicle-status-solar"
			>
				<shopicon-regular-sun></shopicon-regular-sun>
				{{ fmtPercentage(sessionSolarPercentage) }}
			</div>

			<!-- vehicle -->
			<div
				v-if="vehicleClimaterActive"
				class="entry text-primary"
				data-testid="vehicle-status-climater"
			>
				<ClimaterIcon /> on
			</div>
			<div
				v-if="minSocVisible"
				class="entry gap-0 text-danger"
				data-testid="vehicle-status-minsoc"
			>
				<VehicleLimitIcon />
				<shopicon-regular-angledoublerightsmall></shopicon-regular-angledoublerightsmall>
				{{ fmtPercentage(minSoc) }}
			</div>
			<div
				v-else-if="vehicleLimitVisible"
				class="entry"
				:class="vehicleLimitClass"
				data-testid="vehicle-status-limit"
			>
				<component :is="vehicleLimitIconComponent" />
				{{ fmtPercentage(vehicleLimitSoc) }}
			</div>

			<!-- smart cost -->
			<div
				v-if="smartCostVisible"
				class="entry"
				:class="smartCostClass"
				data-testid="vehicle-status-smartcost"
			>
				<DynamicPriceIcon v-if="smartCostPrice" />
				<shopicon-regular-eco1 v-else></shopicon-regular-eco1>
				<div>
					<span v-if="smartCostNowVisible">{{ smartCostNow }}</span>
					≤ {{ smartCostLimitFmt }}
					<span v-if="smartCostNextStart"
						>({{ fmtAbsoluteDate(new Date(smartCostNextStart)) }})</span
					>
				</div>
			</div>

			<!-- plan -->
			<div
				v-if="planEndVisible"
				class="entry text-primary"
				data-testid="vehicle-status-planactive"
			>
				<PlanEndIcon />
				{{ fmtAbsoluteDate(new Date(effectivePlanTime)) }}
			</div>
			<div v-else-if="planStartVisible" class="entry" data-testid="vehicle-status-planstart">
				<PlanStartIcon />
				{{ fmtAbsoluteDate(new Date(planProjectedStart)) }}
			</div>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/angledoublerightsmall";
import DynamicPriceIcon from "./MaterialIcon/DynamicPrice.vue";
import { DEFAULT_LOCALE } from "../i18n";
import formatter from "../mixins/formatter";
import { CO2_TYPE } from "../units";
import PlanStartIcon from "./MaterialIcon/PlanStart.vue";
import PlanEndIcon from "./MaterialIcon/PlanEnd.vue";
import ClimaterIcon from "./MaterialIcon/Climater.vue";
import VehicleLimitReachedIcon from "./MaterialIcon/VehicleLimitReached.vue";
import VehicleLimitWarningIcon from "./MaterialIcon/VehicleLimitWarning.vue";
import VehicleLimitIcon from "./MaterialIcon/VehicleLimit.vue";
import SunDownIcon from "./MaterialIcon/SunDown.vue";
import SunUpIcon from "./MaterialIcon/SunUp.vue";

export default {
	name: "VehicleStatus",
	components: {
		DynamicPriceIcon,
		PlanStartIcon,
		PlanEndIcon,
		ClimaterIcon,
		VehicleLimitIcon,
		SunDownIcon,
		SunUpIcon,
	},
	mixins: [formatter],
	props: {
		vehicleSoc: Number,
		vehicleLimitSoc: Number,
		minSoc: Number,
		enabled: Boolean,
		connected: Boolean,
		charging: Boolean,
		heating: Boolean,
		effectivePlanTime: String,
		effectiveLimitSoc: Number,
		planProjectedStart: String,
		smartCostNextStart: String,
		chargingPlanDisabled: Boolean,
		smartCostDisabled: Boolean,
		planActive: Boolean,
		effectivePlanSoc: Number,
		phaseAction: String,
		phaseRemainingInterpolated: Number,
		greenShareLoadpoints: Number,
		pvAction: String,
		pvRemainingInterpolated: Number,
		vehicleClimaterActive: Boolean,
		sessionSolarPercentage: Number,
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
		pvTimerVisible() {
			return this.pvTimerActive;
		},
		pvTimerClass() {
			return this.pvAction === "disable" ? "text-primary" : "";
		},
		phaseTimerVisible() {
			return !this.pvTimerActive && this.charging && this.phaseTimerActive;
		},
		phaseIconClass() {
			return this.phaseAction === "scale1p" ? "phaseUp" : "phaseDown";
		},
		solarPercentageVisible() {
			return this.sessionSolarPercentage > 0;
		},
		solarPercentageClass() {
			return this.charging && this.greenShareLoadpoints > 0 ? "text-primary" : "";
		},
		vehicleLimitVisible() {
			const limit = Math.max(this.vehicleLimitSoc, this.effectiveLimitSoc) || 100;
			return this.vehicleLimitSoc > 0 && this.vehicleLimitSoc <= limit;
		},
		minSocVisible() {
			return this.minSoc > 0 && this.vehicleSoc < this.minSoc;
		},
		vehicleLimitReached() {
			return (
				!this.charging &&
				this.vehicleSoc &&
				this.vehicleLimitSoc &&
				this.vehicleSoc >= this.vehicleLimitSoc - 1
			);
		},
		vehicleLimitWarning() {
			return this.effectivePlanSoc > this.vehicleLimitSoc;
		},
		vehicleLimitClass() {
			if (this.vehicleLimitReached) {
				return "text-primary";
			}
			if (this.vehicleLimitWarning) {
				return "text-warning";
			}
			return "";
		},
		vehicleLimitIconComponent() {
			if (this.vehicleLimitReached) {
				return VehicleLimitReachedIcon;
			}
			if (this.vehicleLimitWarning) {
				return VehicleLimitWarningIcon;
			}
			return VehicleLimitIcon;
		},
		planStartVisible() {
			return this.planProjectedStart && !this.planActive && !this.chargingPlanDisabled;
		},
		planEndVisible() {
			return this.effectivePlanTime && this.planActive && !this.chargingPlanDisabled;
		},
		smartCostVisible() {
			return !!this.smartCostLimit;
		},
		smartCostPrice() {
			return this.smartCostType !== "co2";
		},
		smartCostNowVisible() {
			if (this.smartCostPrice) {
				return this.tariffGrid <= this.smartCostLimit;
			}
			return this.tariffCo2 <= this.smartCostLimit;
		},
		smartCostNow() {
			if (this.smartCostPrice) {
				return this.fmtPricePerKWh(this.tariffGrid, this.currency, true);
			}
			return this.fmtCo2Short(this.tariffCo2);
		},
		smartCostLimitFmt() {
			if (this.smartCostPrice) {
				return this.fmtPricePerKWh(this.smartCostLimit, this.currency, true);
			}
			return this.fmtCo2Short(this.smartCostLimit);
		},
		smartCostClass() {
			if (this.smartCostDisabled) {
				return "opacity-25";
			}
			if (this.smartCostActive) {
				return "text-primary";
			}
			return "";
		},
		chargerStatus() {
			const t = (key, data) => {
				if (this.heating) {
					// check for special heating status translation
					const name = `main.heatingStatus.${key}`;
					if (this.$te(name, DEFAULT_LOCALE)) {
						return this.$t(name, data);
					}
				}
				return this.$t(`main.vehicleStatus.${key}`, data);
			};

			if (!this.connected) {
				return t("disconnected");
			}

			if (this.enabled && !this.charging) {
				if (this.vehicleLimitReached) {
					return t("finished");
				}
				return t("waitForVehicle");
			}

			if (this.charging) {
				return t("charging");
			}

			return t("connected");
		},
		message: function () {
			const t = (key, data) => {
				if (this.heating) {
					// check for special heating status translation
					const name = `main.heatingStatus.${key}`;
					if (this.$te(name, DEFAULT_LOCALE)) {
						return this.$t(name, data);
					}
				}
				return this.$t(`main.vehicleStatus.${key}`, data);
			};

			if (!this.connected) {
				return t("disconnected");
			}
			// min charge active
			if (this.minSoc > 0 && this.vehicleSoc < this.minSoc) {
				return t("minCharge", { soc: this.fmtPercentage(this.minSoc) });
			}

			// plan
			if (!this.chargingPlanDisabled && this.effectivePlanTime) {
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
				if (this.vehicleLimitSoc > 0 && this.vehicleSoc >= this.vehicleLimitSoc - 1) {
					return t("vehicleLimitReached", {
						soc: this.fmtPercentage(this.vehicleLimitSoc),
					});
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

			if (this.charging) {
				return t("charging");
			}

			return t("connected");
		},
	},
};
</script>

<style scoped>
.entry {
	display: flex;
	align-items: center;
	flex-wrap: nowrap;
	text-wrap: nowrap;
	gap: 0.5rem;
}
.phaseUp {
	transform: rotate(90deg);
}
.phaseDown {
	transform: rotate(-90deg);
}
.tabular {
	font-variant-numeric: tabular-nums;
}
</style>
