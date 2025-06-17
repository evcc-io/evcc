<template>
	<div
		class="d-flex justify-content-between gap-3 evcc-gray align-items-start flex-wrap"
		style="min-height: 24px"
		data-testid="vehicle-status"
	>
		<div class="charger-status" data-testid="vehicle-status-charger">
			{{ chargerStatus }}
		</div>
		<div class="d-flex flex-wrap justify-content-end gap-3 flex-grow-1">
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

			<template v-if="heating">
				<div
					v-if="tempLimitVisible"
					ref="tempLimit"
					class="entry"
					data-bs-toggle="tooltip"
					data-testid="vehicle-status-limit"
				>
					<TempLimitIcon />
					{{ fmtTemperature(vehicleLimitSoc) }}
				</div>
			</template>
			<template v-else>
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
					:role="vehicleLimitWarning ? 'button' : undefined"
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
				<div
					v-if="vehicleWelcomeActive"
					ref="vehicleWelcome"
					data-bs-toggle="tooltip"
					class="entry"
					data-testid="vehicle-status-welcome"
				>
					<WelcomeIcon />
				</div>
				<div
					v-if="awaitingAuthorizationVisible"
					ref="awaitingAuthorization"
					data-bs-toggle="tooltip"
					class="entry"
					data-testid="vehicle-status-awaiting-authorization"
				>
					<RfidWaitIcon />
				</div>
				<div
					v-if="disconnectRequiredVisible"
					ref="disconnectRequired"
					class="entry text-warning"
					data-bs-toggle="tooltip"
					data-testid="vehicle-status-disconnect-required"
				>
					<ReconnectIcon />
				</div>
			</template>

			<!-- smart cost -->
			<button
				v-if="smartCostVisible"
				ref="smartCost"
				type="button"
				class="entry"
				:class="smartCostClass"
				data-testid="vehicle-status-smartcost"
				data-bs-toggle="tooltip"
				data-bs-trigger="hover"
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

			<!-- battery boost -->
			<button
				v-if="batteryBoostVisible"
				ref="batteryBoost"
				type="button"
				class="entry"
				data-testid="vehicle-status-batteryboost"
				data-bs-toggle="tooltip"
				@click="batteryBoostClicked"
			>
				<BatteryBoostIcon />
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
					{{ planProjectedEnd ? fmtAbsoluteDate(new Date(planProjectedEnd)) : "" }}
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
				{{ planProjectedStart ? fmtAbsoluteDate(new Date(planProjectedStart)) : "" }}
			</button>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/angledoublerightsmall";
import "@h2d2/shopicons/es/regular/clock";
import DynamicPriceIcon from "../MaterialIcon/DynamicPrice.vue";
import { DEFAULT_LOCALE } from "@/i18n";
import formatter from "@/mixins/formatter";
import { CO2_TYPE } from "@/units";
import ClimaterIcon from "../MaterialIcon/Climater.vue";
import PlanEndIcon from "../MaterialIcon/PlanEnd.vue";
import PlanStartIcon from "../MaterialIcon/PlanStart.vue";
import ReconnectIcon from "../MaterialIcon/Reconnect.vue";
import RfidWaitIcon from "../MaterialIcon/RfidWait.vue";
import SunDownIcon from "../MaterialIcon/SunDown.vue";
import SunUpIcon from "../MaterialIcon/SunUp.vue";
import TempLimitIcon from "../MaterialIcon/TempLimit.vue";
import Tooltip from "bootstrap/js/dist/tooltip";
import VehicleLimitIcon from "../MaterialIcon/VehicleLimit.vue";
import VehicleLimitReachedIcon from "../MaterialIcon/VehicleLimitReached.vue";
import VehicleLimitWarningIcon from "../MaterialIcon/VehicleLimitWarning.vue";
import VehicleMinSocIcon from "../MaterialIcon/VehicleMinSoc.vue";
import WelcomeIcon from "../MaterialIcon/Welcome.vue";
import BatteryBoostIcon from "../MaterialIcon/BatteryBoost.vue";
import { defineComponent, type PropType } from "vue";
import type { CURRENCY, Timeout } from "@/types/evcc";
const REASON_AUTH = "waitingforauthorization";
const REASON_DISCONNECT = "disconnectrequired";

export default defineComponent({
	name: "VehicleStatus",
	components: {
		ClimaterIcon,
		DynamicPriceIcon,
		PlanEndIcon,
		PlanStartIcon,
		ReconnectIcon,
		RfidWaitIcon,
		SunDownIcon,
		SunUpIcon,
		VehicleLimitIcon,
		VehicleMinSocIcon,
		WelcomeIcon,
		TempLimitIcon,
		BatteryBoostIcon,
	},
	mixins: [formatter],
	props: {
		vehicleSoc: { type: Number, default: 0 },
		batteryBoostActive: Boolean,
		charging: Boolean,
		chargingPlanDisabled: Boolean,
		chargerStatusReason: String,
		connected: Boolean,
		currency: String as PropType<CURRENCY>,
		effectiveLimitSoc: Number,
		effectivePlanSoc: { type: Number, default: 0 },
		effectivePlanTime: String,
		enabled: Boolean,
		heating: Boolean,
		minSoc: { type: Number, default: 0 },
		phaseAction: { type: String, default: "" },
		phaseRemainingInterpolated: Number,
		planActive: Boolean,
		planOverrun: Number,
		planProjectedEnd: String,
		planProjectedStart: String,
		planTimeUnreachable: Boolean,
		pvAction: { type: String, default: "" },
		pvRemainingInterpolated: Number,
		smartCostActive: Boolean,
		smartCostDisabled: Boolean,
		smartCostLimit: { type: Number, default: null },
		smartCostNextStart: String,
		smartCostType: String,
		tariffCo2: { type: Number, default: 0 },
		tariffGrid: { type: Number, default: 0 },
		vehicleClimaterActive: Boolean,
		vehicleWelcomeActive: Boolean,
		vehicleLimitSoc: { type: Number, default: 0 },
	},
	emits: ["open-loadpoint-settings", "open-minsoc-settings", "open-plan-modal"],
	data() {
		return {
			pvTooltip: null as Tooltip | null,
			phaseTooltip: null as Tooltip | null,
			planStartTooltip: null as Tooltip | null,
			planActiveTooltip: null as Tooltip | null,
			minSocTooltip: null as Tooltip | null,
			vehicleClimaterTooltip: null as Tooltip | null,
			vehicleWelcomeTooltip: null as Tooltip | null,
			smartCostTooltip: null as Tooltip | null,
			vehicleLimitTooltip: null as Tooltip | null,
			tempLimitTooltip: null as Tooltip | null,
			awaitingAuthorizationTooltip: null as Tooltip | null,
			disconnectRequiredTooltip: null as Tooltip | null,
			batteryBoostTooltip: null as Tooltip | null,
			interval: null as Timeout,
			planProjectedEndDuration: null as string | null,
			smartCostNextStartDuration: null as string | null,
			planProjectedStartDuration: null as string | null,
		};
	},
	computed: {
		phaseTimerActive() {
			return (
				this.phaseRemainingInterpolated &&
				this.phaseRemainingInterpolated > 0 &&
				["scale1p", "scale3p"].includes(this.phaseAction)
			);
		},
		pvTimerActive() {
			return (
				this.pvRemainingInterpolated &&
				this.pvRemainingInterpolated > 0 &&
				["enable", "disable"].includes(this.pvAction)
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
		awaitingAuthorizationVisible() {
			return this.chargerStatusReason === REASON_AUTH;
		},
		awaitingAuthorizationTooltipContent() {
			if (!this.awaitingAuthorizationVisible) {
				return "";
			}
			return this.$t("main.vehicleStatus.awaitingAuthorization");
		},
		disconnectRequiredVisible() {
			return this.chargerStatusReason === REASON_DISCONNECT;
		},
		disconnectRequiredTooltipContent() {
			if (!this.disconnectRequiredVisible) {
				return "";
			}
			return this.$t("main.vehicleStatus.disconnectRequired");
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
		tempLimitTooltipContent() {
			if (!this.tempLimitVisible) {
				return "";
			}
			return this.$t("main.heatingStatus.vehicleLimit");
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
		tempLimitVisible() {
			return this.heating && this.vehicleLimitSoc > 0;
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
		batteryBoostVisible() {
			return this.batteryBoostActive;
		},
		batteryBoostTooltipContent() {
			if (!this.batteryBoostVisible) {
				return "";
			}
			return this.$t("main.vehicleStatus.batteryBoost");
		},
		planStartTooltipContent() {
			if (!this.planStartVisible) {
				return "";
			}
			return this.$t("main.vehicleStatus.targetChargePlanned", {
				duration: this.planProjectedStartDuration,
			});
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
				duration: this.planProjectedEndDuration,
			});
		},
		smartCostVisible() {
			return this.smartCostLimit !== null;
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
					duration: this.smartCostNextStartDuration,
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
			if (!this.vehicleClimaterActive) {
				return "";
			}
			return this.$t("main.vehicleStatus.climating");
		},
		vehicleWelcomeTooltipContent() {
			if (!this.vehicleWelcomeActive) {
				return "";
			}
			return this.$t("main.vehicleStatus.welcome");
		},
		chargerStatus() {
			const t = (key: string) => {
				if (this.heating) {
					// check for special heating status translation
					const name = `main.heatingStatus.${key}`;
					if (this.$te(name, DEFAULT_LOCALE)) {
						return this.$t(name);
					}
				}
				return this.$t(`main.vehicleStatus.${key}`);
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
		vehicleWelcomeTooltipContent() {
			this.$nextTick(this.updateVehicleWelcomeTooltip);
		},
		smartCostTooltipContent() {
			this.$nextTick(this.updateSmartCostTooltip);
		},
		vehicleLimitTooltipContent() {
			this.$nextTick(this.updateVehicleLimitTooltip);
		},
		tempLimitTooltipContent() {
			this.$nextTick(this.updateTempLimitTooltip);
		},
		awaitingAuthorizationTooltipContent() {
			this.$nextTick(this.updateAwaitingAuthorizationTooltip);
		},
		disconnectRequiredTooltipContent() {
			this.$nextTick(this.updateDisconnectRequiredTooltip);
		},
		batteryBoostTooltipContent() {
			this.$nextTick(this.updateBatteryBoostTooltip);
		},
		planProjectedStart() {
			this.updateDurations();
		},
		planProjectedEnd() {
			this.updateDurations();
		},
		smartCostNextStart() {
			this.updateDurations();
		},
	},
	mounted() {
		this.updatePlanStartTooltip();
		this.updatePlanActiveTooltip();
		this.updateMinSocTooltip();
		this.updatePhaseTooltip();
		this.updatePvTooltip();
		this.updateVehicleClimaterTooltip();
		this.updateVehicleWelcomeTooltip();
		this.updateSmartCostTooltip();
		this.updateVehicleLimitTooltip();
		this.updateTempLimitTooltip();
		this.updateAwaitingAuthorizationTooltip();
		this.updateDisconnectRequiredTooltip();
		this.updateDurations();

		this.interval = setInterval(this.updateDurations, 1000 * 60);
		this.updateBatteryBoostTooltip();
	},
	beforeUnmount() {
		if (this.interval) {
			clearInterval(this.interval);
		}
	},
	methods: {
		updateDurations() {
			if (this.planProjectedStart) {
				this.planProjectedStartDuration = this.fmtDurationToTime(
					new Date(this.planProjectedStart)
				);
			}
			if (this.planProjectedEnd) {
				this.planProjectedEndDuration = this.fmtDurationToTime(
					new Date(this.planProjectedEnd)
				);
			}
			if (this.smartCostNextStart) {
				this.smartCostNextStartDuration = this.fmtDurationToTime(
					new Date(this.smartCostNextStart)
				);
			}
		},
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
		batteryBoostClicked() {
			this.batteryBoostTooltip?.hide();
			this.openLoadpointSettings();
		},
		updatePvTooltip() {
			this.pvTooltip = this.updateTooltip(
				this.pvTooltip,
				this.pvTimerContent,
				this.$refs["pvTimer"]
			);
		},
		updatePhaseTooltip() {
			this.phaseTooltip = this.updateTooltip(
				this.phaseTooltip,
				this.phaseTimerContent,
				this.$refs["phaseTimer"]
			);
		},
		updatePlanStartTooltip() {
			this.planStartTooltip = this.updateTooltip(
				this.planStartTooltip,
				this.planStartTooltipContent,
				this.$refs["planStart"],
				true
			);
		},
		updatePlanActiveTooltip() {
			this.planActiveTooltip = this.updateTooltip(
				this.planActiveTooltip,
				this.planActiveTooltipContent,
				this.$refs["planActive"],
				true
			);
		},
		updateMinSocTooltip() {
			this.minSocTooltip = this.updateTooltip(
				this.minSocTooltip,
				this.minSocTooltipContent,
				this.$refs["minSoc"],
				true
			);
		},
		updateVehicleClimaterTooltip() {
			this.vehicleClimaterTooltip = this.updateTooltip(
				this.vehicleClimaterTooltip,
				this.vehicleClimaterTooltipContent,
				this.$refs["vehicleClimater"]
			);
		},
		updateBatteryBoostTooltip() {
			this.batteryBoostTooltip = this.updateTooltip(
				this.batteryBoostTooltip,
				this.batteryBoostTooltipContent,
				this.$refs["batteryBoost"]
			);
		},
		updateVehicleWelcomeTooltip() {
			this.vehicleWelcomeTooltip = this.updateTooltip(
				this.vehicleWelcomeTooltip,
				this.vehicleWelcomeTooltipContent,
				this.$refs["vehicleWelcome"]
			);
		},
		updateSmartCostTooltip() {
			this.smartCostTooltip = this.updateTooltip(
				this.smartCostTooltip,
				this.smartCostTooltipContent,
				this.$refs["smartCost"],
				true
			);
		},
		updateVehicleLimitTooltip() {
			this.vehicleLimitTooltip = this.updateTooltip(
				this.vehicleLimitTooltip,
				this.vehicleLimitTooltipContent,
				this.$refs["vehicleLimit"],
				true
			);
		},
		updateTempLimitTooltip() {
			this.tempLimitTooltip = this.updateTooltip(
				this.tempLimitTooltip,
				this.tempLimitTooltipContent,
				this.$refs["tempLimit"]
			);
		},
		updateAwaitingAuthorizationTooltip() {
			this.awaitingAuthorizationTooltip = this.updateTooltip(
				this.awaitingAuthorizationTooltip,
				this.awaitingAuthorizationTooltipContent,
				this.$refs["awaitingAuthorization"]
			);
		},
		updateDisconnectRequiredTooltip() {
			this.disconnectRequiredTooltip = this.updateTooltip(
				this.disconnectRequiredTooltip,
				this.disconnectRequiredTooltipContent,
				this.$refs["disconnectRequired"]
			);
		},
		updateTooltip(
			instance: Tooltip | null,
			content: string,
			ref: HTMLElement | undefined,
			hoverOnly = false
		): Tooltip | null {
			if (!content || !ref) {
				if (instance) {
					instance.dispose();
				}
				return null;
			}
			let newInstance = instance;
			if (!newInstance) {
				const trigger = hoverOnly ? "hover" : "hover focus";
				newInstance = new Tooltip(ref, { title: " ", trigger });
			}
			newInstance.setContent({ ".tooltip-inner": content });
			return newInstance;
		},
	},
});
</script>

<style scoped>
.charger-status {
	padding-top: 2px;
}
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
