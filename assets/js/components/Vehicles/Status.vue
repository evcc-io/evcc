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
			<StatusItem
				v-for="item in statusItems"
				:key="item.id"
				:content="item.content"
				:tooltip-content="item.tooltipContent"
				:icon-component="item.iconComponent"
				:icon-class="item.iconClass"
				:data-testid="item.testId"
				:class="item.itemClass"
				:clickable="item.clickable"
				:tabular="item.tabular"
				@click="item.clickHandler"
			>
				<!-- items with complex content -->
				<template v-if="item.id === 'smartCost'" #default>
					<div>
						<span v-if="smartCostNowVisible">{{ smartCostNow }}</span>
						≤ <span class="text-decoration-underline">{{ smartCostLimitFmt }}</span>
						<span v-if="smartCostNextStart">
							({{ fmtAbsoluteDate(new Date(smartCostNextStart)) }})
						</span>
					</div>
				</template>
				<template v-else-if="item.id === 'smartFeedInPriority'" #default>
					<div>
						<span v-if="smartFeedInPriorityActive">{{ feedInNow }}</span>
						≥
						<span class="text-decoration-underline">{{
							smartFeedInPriorityLimitFmt
						}}</span>
						<span v-if="smartFeedInPriorityNextStart">
							({{ fmtAbsoluteDate(new Date(smartFeedInPriorityNextStart)) }})
						</span>
					</div>
				</template>
			</StatusItem>
		</div>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/eco1";
import "@h2d2/shopicons/es/regular/angledoublerightsmall";
import "@h2d2/shopicons/es/regular/clock";
import { DEFAULT_LOCALE } from "@/i18n.ts";
import formatter from "@/mixins/formatter";
import { defineComponent, type PropType } from "vue";
import { SMART_COST_TYPE, type CURRENCY, type Timeout } from "@/types/evcc";

import ClimaterIcon from "../MaterialIcon/Climater.vue";
import DynamicPriceIcon from "../MaterialIcon/DynamicPrice.vue";
import PlanEndIcon from "../MaterialIcon/PlanEnd.vue";
import PlanStartIcon from "../MaterialIcon/PlanStart.vue";
import ReconnectIcon from "../MaterialIcon/Reconnect.vue";
import RfidWaitIcon from "../MaterialIcon/RfidWait.vue";
import SunDownIcon from "../MaterialIcon/SunDown.vue";
import SunUpIcon from "../MaterialIcon/SunUp.vue";
import TempLimitIcon from "../MaterialIcon/TempLimit.vue";
import VehicleLimitIcon from "../MaterialIcon/VehicleLimit.vue";
import VehicleLimitReachedIcon from "../MaterialIcon/VehicleLimitReached.vue";
import VehicleLimitWarningIcon from "../MaterialIcon/VehicleLimitWarning.vue";
import VehicleMinSocIcon from "../MaterialIcon/VehicleMinSoc.vue";
import WelcomeIcon from "../MaterialIcon/Welcome.vue";
import BatteryBoostIcon from "../MaterialIcon/BatteryBoost.vue";
import SunPauseIcon from "../MaterialIcon/SunPause.vue";

import StatusItem from "./StatusItem.vue";

const REASON_AUTH = "waitingforauthorization";
const REASON_DISCONNECT = "disconnectrequired";

export default defineComponent({
	name: "VehicleStatus",
	components: {
		StatusItem,
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
		smartFeedInPriorityActive: Boolean,
		smartFeedInPriorityDisabled: Boolean,
		smartFeedInPriorityLimit: { type: Number, default: null },
		smartFeedInPriorityNextStart: String,
		tariffCo2: { type: Number, default: 0 },
		tariffGrid: { type: Number, default: 0 },
		tariffFeedIn: { type: Number, default: 0 },
		vehicleClimaterActive: Boolean,
		vehicleWelcomeActive: Boolean,
		vehicleLimitSoc: { type: Number, default: 0 },
	},
	emits: ["open-loadpoint-settings", "open-minsoc-settings", "open-plan-modal"],
	data() {
		return {
			interval: null as Timeout,
		};
	},
	computed: {
		pvTimerActive() {
			return (
				this.pvRemainingInterpolated &&
				this.pvRemainingInterpolated > 0 &&
				["enable", "disable"].includes(this.pvAction)
			);
		},
		phaseTimerActive() {
			return (
				this.phaseRemainingInterpolated &&
				this.phaseRemainingInterpolated > 0 &&
				["scale1p", "scale3p"].includes(this.phaseAction)
			);
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
		smartCostPrice() {
			return this.smartCostType !== SMART_COST_TYPE.CO2;
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
		feedInNow() {
			return this.fmtPricePerKWh(this.tariffFeedIn, this.currency, true);
		},
		smartFeedInPriorityLimitFmt() {
			return this.fmtPricePerKWh(this.smartFeedInPriorityLimit, this.currency, true);
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
		statusItems() {
			const t = (key: string, params?: Record<string, unknown>) =>
				this.$t(`main.vehicleStatus.${key}`, params ?? {});

			const items = [
				{
					id: "pvTimer",
					visible: Boolean(this.pvTimerActive),
					content: this.fmtDuration(this.pvRemainingInterpolated),
					tooltipContent: this.pvAction === "enable" ? t("pvEnable") : t("pvDisable"),
					iconComponent: this.pvAction === "enable" ? SunUpIcon : SunDownIcon,
					testId: "vehicle-status-pvtimer",
					tabular: true,
				},
				{
					id: "phaseTimer",
					visible: !this.pvTimerActive && this.charging && this.phaseTimerActive,
					content: this.fmtDuration(this.phaseRemainingInterpolated),
					tooltipContent: t(this.phaseAction),
					iconComponent: "shopicon-regular-angledoublerightsmall",
					iconClass: this.phaseAction === "scale1p" ? "phaseUp" : "phaseDown",
					testId: "vehicle-status-phasetimer",
					tabular: true,
				},
				{
					id: "tempLimit",
					visible: this.heating && this.vehicleLimitSoc > 0,
					content: this.fmtTemperature(this.vehicleLimitSoc),
					tooltipContent: t("vehicleLimit"),
					iconComponent: TempLimitIcon,
					testId: "vehicle-status-limit",
				},
				{
					id: "minSoc",
					visible:
						!this.heating &&
						this.connected &&
						this.minSoc > 0 &&
						this.vehicleSoc < this.minSoc,
					content: this.fmtPercentage(this.minSoc),
					tooltipContent: t("minCharge", {
						soc: this.fmtPercentage(this.minSoc),
					}),
					iconComponent: VehicleMinSocIcon,
					itemClass: "text-danger text-decoration-underline",
					testId: "vehicle-status-minsoc",
					clickable: true,
					clickHandler: () => this.openMinSocSettings(),
				},
				{
					id: "vehicleLimit",
					visible:
						!this.heating &&
						this.connected &&
						this.vehicleLimitSoc > 0 &&
						this.vehicleLimitSoc < (this.effectiveLimitSoc || 100),
					content: this.fmtPercentage(this.vehicleLimitSoc),
					tooltipContent: this.vehicleLimitReached
						? t("vehicleLimitReached")
						: this.vehicleLimitWarning
							? this.$t("main.targetCharge.targetIsAboveVehicleLimit")
							: t("vehicleLimit"),
					iconComponent: this.vehicleLimitReached
						? VehicleLimitReachedIcon
						: this.vehicleLimitWarning
							? VehicleLimitWarningIcon
							: VehicleLimitIcon,
					itemClass: this.vehicleLimitWarning ? "text-warning" : "",
					testId: "vehicle-status-limit",
					clickable: Boolean(this.vehicleLimitWarning),
					clickHandler: this.vehicleLimitWarning ? () => this.openPlanModal() : undefined,
				},
				{
					id: "vehicleClimater",
					visible: !this.heating && this.vehicleClimaterActive,
					tooltipContent: t("climating"),
					iconComponent: ClimaterIcon,
					testId: "vehicle-status-climater",
				},
				{
					id: "vehicleWelcome",
					visible: !this.heating && this.vehicleWelcomeActive,
					tooltipContent: t("welcome"),
					iconComponent: WelcomeIcon,
					testId: "vehicle-status-welcome",
				},
				{
					id: "awaitingAuthorization",
					visible: !this.heating && this.chargerStatusReason === REASON_AUTH,
					tooltipContent: t("awaitingAuthorization"),
					iconComponent: RfidWaitIcon,
					testId: "vehicle-status-awaiting-authorization",
				},
				{
					id: "disconnectRequired",
					visible: !this.heating && this.chargerStatusReason === REASON_DISCONNECT,
					tooltipContent: t("disconnectRequired"),
					iconComponent: ReconnectIcon,
					itemClass: "text-warning",
					testId: "vehicle-status-disconnect-required",
				},
				{
					id: "smartCost",
					visible: this.smartCostLimit !== null,
					tooltipContent: this.getSmartCostTooltip(),
					iconComponent: this.smartCostPrice ? DynamicPriceIcon : "shopicon-regular-eco1",
					itemClass: this.smartCostDisabled
						? "opacity-25"
						: this.smartCostActive
							? "text-primary"
							: "",
					testId: "vehicle-status-smartcost",
					clickable: true,
					clickHandler: () => this.openLoadpointSettings(),
				},
				{
					id: "smartFeedInPriority",
					visible: this.smartFeedInPriorityActive || this.smartFeedInPriorityNextStart,
					tooltipContent: this.getSmartFeedInPriorityTooltip(),
					iconComponent: SunPauseIcon,
					itemClass: this.smartFeedInPriorityDisabled
						? "opacity-25"
						: this.smartFeedInPriorityActive
							? "text-warning"
							: "",
					testId: "vehicle-status-smartfeedinpriority",
					clickable: true,
					clickHandler: () => this.openLoadpointSettings(),
				},
				{
					id: "batteryBoost",
					visible: this.batteryBoostActive,
					tooltipContent: t("batteryBoost"),
					iconComponent: BatteryBoostIcon,
					testId: "vehicle-status-batteryboost",
					clickable: true,
					clickHandler: () => this.openLoadpointSettings(),
				},
				{
					id: "planActive",
					visible: this.planProjectedEnd && this.planActive && !this.chargingPlanDisabled,
					content: this.planProjectedEnd
						? this.fmtAbsoluteDate(new Date(this.planProjectedEnd))
						: "",
					tooltipContent: this.planTimeUnreachable
						? this.$t("main.targetCharge.notReachableInTime", {
								overrun: this.fmtDuration(this.planOverrun, true, "h"),
							})
						: t("targetChargeActive", {
								duration: this.planProjectedEnd
									? this.fmtDurationToTime(new Date(this.planProjectedEnd))
									: "",
							}),
					iconComponent: PlanEndIcon,
					itemClass: this.planTimeUnreachable ? "text-warning" : "text-primary",
					testId: "vehicle-status-planactive",
					clickable: true,
					clickHandler: () => this.openPlanModal(),
				},
				{
					id: "planStart",
					visible:
						this.planProjectedStart && !this.planActive && !this.chargingPlanDisabled,
					content: this.planProjectedStart
						? this.fmtAbsoluteDate(new Date(this.planProjectedStart))
						: "",
					tooltipContent: t("targetChargePlanned", {
						duration: this.planProjectedStart
							? this.fmtDurationToTime(new Date(this.planProjectedStart))
							: "",
					}),
					iconComponent: PlanStartIcon,
					testId: "vehicle-status-planstart",
					clickable: true,
					clickHandler: () => this.openPlanModal(),
				},
			];

			return items.filter((item) => item.visible);
		},
	},
	mounted() {
		this.interval = setInterval(() => {
			// Force reactivity update for time-dependent content
			this.$forceUpdate();
		}, 1000 * 60);
	},
	beforeUnmount() {
		if (this.interval) {
			clearInterval(this.interval);
		}
	},
	methods: {
		openLoadpointSettings() {
			this.$emit("open-loadpoint-settings");
		},
		openMinSocSettings() {
			this.$emit("open-minsoc-settings");
		},
		openPlanModal() {
			this.$emit("open-plan-modal");
		},
		getSmartCostTooltip() {
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
		getSmartFeedInPriorityTooltip() {
			const prefix = "main.vehicleStatus.feedinPriority";
			if (this.smartFeedInPriorityActive) {
				return this.$t(`${prefix}Pausing`);
			}
			if (this.smartFeedInPriorityNextStart) {
				return this.$t(`${prefix}NextStart`, {
					duration: this.fmtDurationToTime(new Date(this.smartFeedInPriorityNextStart)),
				});
			}
			return "";
		},
	},
});
</script>

<style scoped>
.charger-status {
	padding-top: 2px;
}

.tabular {
	font-variant-numeric: tabular-nums;
}
</style>
