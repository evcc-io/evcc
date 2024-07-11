<template>
	<div
		class="d-flex justify-content-between gap-4 evcc-gray"
		style="min-height: 24px"
		data-testid="vehicle-status"
	>
		<div class="text-nowrap" data-testid="vehicle-status-charger">{{ chargerStatus }}</div>
		<div class="d-flex flex-wrap justify-content-end gap-3">
			<!-- pv/phase timer -->
			<div
				v-if="pvTimerVisible"
				ref="pvTimer"
				class="entry"
				data-testid="vehicle-status-pvtimer"
				data-bs-toggle="tooltip"
			>
				<SunUpIcon v-if="pvAction === 'enable'" />
				<SunDownIcon v-else />
				<div class="tabular">{{ fmtDuration(pvRemainingInterpolated) }}</div>
			</div>
			<div
				v-else-if="phaseTimerVisible"
				ref="phaseTimer"
				class="entry"
				data-testid="vehicle-status-phasetimer"
				data-bs-toggle="tooltip"
			>
				<shopicon-regular-angledoublerightsmall
					:class="phaseIconClass"
				></shopicon-regular-angledoublerightsmall>
				<div class="tabular">{{ fmtDuration(phaseRemainingInterpolated) }}</div>
			</div>

			<!-- vehicle -->
			<button
				v-if="minSocVisible"
				ref="minSoc"
				type="button"
				class="entry text-danger text-decoration-underline"
				data-testid="vehicle-status-minsoc"
				data-bs-toggle="tooltip"
				@click="openMinSocSettings"
			>
				<VehicleMinSocIcon />
				{{ fmtPercentage(minSoc) }}
			</button>
			<div
				v-else-if="vehicleLimitVisible"
				ref="vehicleLimit"
				class="entry"
				:class="vehicleLimitClass"
				data-bs-toggle="tooltip"
				data-testid="vehicle-status-limit"
				:role="vehicleLimitWarning ? 'button' : null"
				@click="vehicleLimitClicked"
			>
				<component :is="vehicleLimitIconComponent" />
				{{ fmtPercentage(vehicleLimitSoc) }}
			</div>
			<div
				v-if="vehicleClimaterActive"
				ref="vehicleClimater"
				data-bs-toggle="tooltip"
				class="entry"
				data-testid="vehicle-status-climater"
			>
				<ClimaterIcon />
			</div>

			<!-- smart cost -->
			<button
				v-if="smartCostVisible"
				ref="smartCost"
				type="button"
				class="entry"
				:class="smartCostClass"
				data-testid="vehicle-status-smartcost"
				data-bs-toggle="tooltip"
				@click="smartCostClicked"
			>
				<DynamicPriceIcon v-if="smartCostPrice" />
				<shopicon-regular-eco1 v-else></shopicon-regular-eco1>
				<div>
					<span v-if="smartCostNowVisible">{{ smartCostNow }}</span>
					≤ <span class="text-decoration-underline">{{ smartCostLimitFmt }}</span>
					<span v-if="smartCostNextStart">
						({{ fmtAbsoluteDate(new Date(smartCostNextStart)) }})
					</span>
				</div>
			</button>

			<!-- plan -->
			<button
				v-if="planActiveVisible"
				ref="planActive"
				type="button"
				class="entry"
				:class="planActiveClass"
				data-testid="vehicle-status-planactive"
				data-bs-toggle="tooltip"
				@click="planActiveClicked"
			>
				<PlanEndIcon />
				<span>
					{{ fmtAbsoluteDate(new Date(planProjectedEnd)) }}
				</span>
			</button>
			<button
				v-else-if="planStartVisible"
				ref="planStart"
				type="button"
				class="entry"
				data-testid="vehicle-status-planstart"
				data-bs-toggle="tooltip"
				@click="planStartClicked"
			>
				<PlanStartIcon />
				{{ fmtAbsoluteDate(new Date(planProjectedStart)) }}
			</button>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/angledoublerightsmall";
import "@h2d2/shopicons/es/regular/clock";
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
import VehicleMinSocIcon from "./MaterialIcon/VehicleMinSoc.vue";
import SunDownIcon from "./MaterialIcon/SunDown.vue";
import SunUpIcon from "./MaterialIcon/SunUp.vue";
import Tooltip from "bootstrap/js/dist/tooltip";

export default {
	name: "VehicleStatus",
	components: {
		DynamicPriceIcon,
		PlanStartIcon,
		PlanEndIcon,
		ClimaterIcon,
		VehicleLimitIcon,
		VehicleMinSocIcon,
		SunDownIcon,
		SunUpIcon,
	},
	mixins: [formatter],
	props: {
		vehicleSoc: Number,
		charging: Boolean,
		chargingPlanDisabled: Boolean,
		connected: Boolean,
		currency: String,
		effectiveLimitSoc: Number,
		effectivePlanSoc: Number,
		effectivePlanTime: String,
		enabled: Boolean,
		heating: Boolean,
		minSoc: Number,
		phaseAction: String,
		phaseRemainingInterpolated: Number,
		planActive: Boolean,
		planOverrun: Number,
		planProjectedEnd: String,
		planProjectedStart: String,
		planTimeUnreachable: Boolean,
		pvAction: String,
		pvRemainingInterpolated: Number,
		smartCostActive: Boolean,
		smartCostDisabled: Boolean,
		smartCostLimit: Number,
		smartCostNextStart: String,
		smartCostType: String,
		tariffCo2: Number,
		tariffGrid: Number,
		vehicleClimaterActive: Boolean,
		vehicleLimitSoc: Number,
	},
	emits: ["open-loadpoint-settings", "open-minsoc-settings", "open-plan-modal"],
	data() {
		return {
			pvTooltip: null,
			phaseTooltip: null,
			planStartTooltip: null,
			planActiveTooltip: null,
			minSocTooltip: null,
			vehicleClimaterTooltip: null,
			smartCostTooltip: null,
			vehicleLimitTooltip: null,
		};
	},
	mounted() {
		this.updatePlanStartTooltip();
		this.updatePlanActiveTooltip();
		this.updateMinSocTooltip();
		this.updatePhaseTooltip();
		this.updatePvTooltip();
		this.updateVehicleClimaterTooltip();
		this.updateSmartCostTooltip();
		this.updateVehicleLimitTooltip();
	},
	watch: {
		planActiveTooltipContent() {
			this.$nextTick(this.updatePlanActiveTooltip);
		},
		planStartTooltipContent() {
			this.$nextTick(this.updatePlanStartTooltip);
		},
		minSocTooltipContent() {
			this.$nextTick(this.updateMinSocTooltip);
		},
		phaseTimerContent() {
			this.$nextTick(this.updatePhaseTooltip);
		},
		pvTimerContent() {
			this.$nextTick(this.updatePvTooltip);
		},
		vehicleClimaterTooltipContent() {
			this.$nextTick(this.updateVehicleClimaterTooltip);
		},
		smartCostTooltipContent() {
			this.$nextTick(this.updateSmartCostTooltip);
		},
		vehicleLimitTooltipContent() {
			this.$nextTick(this.updateVehicleLimitTooltip);
		},
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
		pvTimerContent() {
			if (!this.pvTimerVisible) {
				return "";
			}
			const key = this.pvAction === "enable" ? "pvEnable" : "pvDisable";
			return this.$t(`main.vehicleStatus.${key}`);
		},
		phaseTimerVisible() {
			return !this.pvTimerActive && this.charging && this.phaseTimerActive;
		},
		phaseIconClass() {
			return this.phaseAction === "scale1p" ? "phaseUp" : "phaseDown";
		},
		phaseTimerContent() {
			if (!this.phaseTimerVisible) {
				return "";
			}
			return this.$t(`main.vehicleStatus.${this.phaseAction}`);
		},
		vehicleLimitVisible() {
			const limit = this.effectiveLimitSoc || 100;
			return this.connected && this.vehicleLimitSoc > 0 && this.vehicleLimitSoc < limit;
		},
		vehicleLimitTooltipContent() {
			if (!this.vehicleLimitVisible) {
				return "";
			}
			if (this.vehicleLimitReached) {
				return this.$t("main.vehicleStatus.vehicleLimitReached");
			}
			if (this.vehicleLimitWarning) {
				return this.$t("main.targetCharge.targetIsAboveVehicleLimit");
			}
			return this.$t("main.vehicleStatus.vehicleLimit");
		},
		minSocVisible() {
			return this.connected && this.minSoc > 0 && this.vehicleSoc < this.minSoc;
		},
		minSocTooltipContent() {
			if (!this.minSocVisible) {
				return "";
			}
			return this.$t("main.vehicleStatus.minCharge", {
				soc: this.fmtPercentage(this.minSoc),
			});
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
		planStartTooltipContent() {
			if (!this.planStartVisible) {
				return "";
			}
			const duration = this.fmtDurationToTime(new Date(this.planProjectedStart));
			return this.$t("main.vehicleStatus.targetChargePlanned", { duration });
		},
		planActiveVisible() {
			return this.planProjectedEnd && this.planActive && !this.chargingPlanDisabled;
		},
		planActiveClass() {
			return this.planTimeUnreachable ? "text-warning" : "text-primary";
		},
		planActiveTooltipContent() {
			if (!this.planActiveVisible) {
				return "";
			}
			if (this.planTimeUnreachable) {
				return this.$t("main.targetCharge.notReachableInTime", {
					overrun: this.fmtDuration(this.planOverrun, true, "h"),
				});
			}
			return this.$t("main.vehicleStatus.targetChargeActive", {
				duration: this.fmtDurationToTime(new Date(this.planProjectedEnd)),
			});
		},
		smartCostVisible() {
			return !!this.smartCostLimit;
		},
		smartCostTooltipContent() {
			if (!this.smartCostVisible) {
				return "";
			}
			const prefix = `main.vehicleStatus.${this.smartCostPrice ? "cheap" : "clean"}`;
			if (this.smartCostNowVisible) {
				return this.$t(`${prefix}EnergyCharging`);
			}
			if (this.smartCostNextStart) {
				return this.$t(`${prefix}EnergyNextStart`, {
					duration: this.fmtDurationToTime(new Date(this.smartCostNextStart)),
				});
			}
			return this.$t(`${prefix}EnergySet`);
		},
		smartCostPrice() {
			return this.smartCostType !== CO2_TYPE;
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
		vehicleClimaterTooltipContent() {
			return this.$t("main.vehicleStatus.climating");
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
	methods: {
		openLoadpointSettings() {
			this.$emit("open-loadpoint-settings");
		},
		openMinSocSettings() {
			this.minSocTooltip?.hide();
			this.$emit("open-minsoc-settings");
		},
		planStartClicked() {
			this.planStartTooltip?.hide();
			this.$emit("open-plan-modal");
		},
		planActiveClicked() {
			this.planActiveTooltip?.hide();
			this.$emit("open-plan-modal");
		},
		vehicleLimitClicked() {
			if (this.vehicleLimitWarning) {
				this.vehicleLimitTooltip?.hide();
				this.$emit("open-plan-modal");
			}
		},
		smartCostClicked() {
			this.openLoadpointSettings();
			this.smartCostTooltip?.hide();
		},
		updatePvTooltip() {
			this.pvTooltip = this.updateTooltip(
				this.pvTooltip,
				this.pvTimerContent,
				this.$refs.pvTimer
			);
		},
		updatePhaseTooltip() {
			this.phaseTooltip = this.updateTooltip(
				this.phaseTooltip,
				this.phaseTimerContent,
				this.$refs.phaseTimer
			);
		},
		updatePlanStartTooltip() {
			this.planStartTooltip = this.updateTooltip(
				this.planStartTooltip,
				this.planStartTooltipContent,
				this.$refs.planStart,
				true
			);
		},
		updatePlanActiveTooltip() {
			this.planActiveTooltip = this.updateTooltip(
				this.planActiveTooltip,
				this.planActiveTooltipContent,
				this.$refs.planActive,
				true
			);
		},
		updateMinSocTooltip() {
			this.minSocTooltip = this.updateTooltip(
				this.minSocTooltip,
				this.minSocTooltipContent,
				this.$refs.minSoc,
				true
			);
		},
		updateVehicleClimaterTooltip() {
			this.vehicleClimaterTooltip = this.updateTooltip(
				this.vehicleClimaterTooltip,
				this.vehicleClimaterTooltipContent,
				this.$refs.vehicleClimater
			);
		},
		updateSmartCostTooltip() {
			this.smartCostTooltip = this.updateTooltip(
				this.smartCostTooltip,
				this.smartCostTooltipContent,
				this.$refs.smartCost,
				true
			);
		},
		updateVehicleLimitTooltip() {
			this.vehicleLimitTooltip = this.updateTooltip(
				this.vehicleLimitTooltip,
				this.vehicleLimitTooltipContent,
				this.$refs.vehicleLimit,
				true
			);
		},
		updateTooltip: function (instance, content, ref, hoverOnly = false) {
			if (!content || !ref) {
				if (instance) {
					instance.dispose();
				}
				return;
			}
			if (!instance) {
				const trigger = hoverOnly ? "hover" : "hover focus";
				instance = new Tooltip(ref, { title: " ", trigger });
			}
			instance.setContent({ ".tooltip-inner": content });
			return instance;
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
	border: none;
	color: inherit;
	background: none;
	padding: 0;
	gap: 0.5rem;
	transition:
		color var(--evcc-transition-medium) linear,
		opacity var(--evcc-transition-medium) linear;
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
