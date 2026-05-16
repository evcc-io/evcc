<template>
	<div class="vehicle pt-4">
		<VehicleTitle
			v-if="!integratedDevice"
			v-bind="vehicleTitleProps"
			@change-vehicle="changeVehicle"
			@remove-vehicle="removeVehicle"
		/>
		<VehicleStatus
			v-bind="vehicleStatus"
			class="mb-2"
			@open-loadpoint-settings="$emit('open-loadpoint-settings')"
			@open-minsoc-settings="openPlanModal(true)"
			@open-plan-modal="openPlanModal"
		/>
		<div class="mt-2 mb-4 d-flex gap-2">
			<BatteryBoostButton
				v-if="showBoostButton"
				class="flex-grow-0"
				v-bind="batteryBoostButtonProps"
				@updated="$emit('batteryboost-updated', $event)"
				@status="handleBoostStatus"
			/>
			<VehicleSoc
				class="flex-grow-1 position-relative"
				v-bind="vehicleSocProps"
				@limit-soc-updated="limitSocUpdated"
				@limit-soc-drag="limitSocDrag"
				@plan-clicked="openPlanModal"
			/>
		</div>
		<div class="details d-flex flex-wrap justify-content-between">
			<LabelAndValue
				v-if="socBasedCharging"
				class="flex-grow-1"
				:label="vehicleSocTitle"
				:value="formattedSoc"
				:extraValue="range ? `${fmtNumber(range, 0)} ${rangeUnit}` : ''"
				data-testid="current-soc"
				align="start"
			/>
			<LabelAndValue
				v-else
				class="flex-grow-1"
				:label="$t('main.loadpoint.charged')"
				:value="fmtEnergy(chargedEnergy)"
				:extraValue="chargedSoc || ''"
				data-testid="current-energy"
				align="start"
			/>
			<ChargingPlan
				v-if="!heating"
				ref="chargingPlan"
				class="flex-grow-1 target-charge"
				v-bind="chargingPlan"
				:disabled="chargingPlanDisabled"
				@open-modal="$emit('open-modal')"
			/>
			<LimitSocSelect
				v-if="socBasedCharging"
				class="flex-grow-1 text-end"
				:limit-soc="displayLimitSoc"
				:range-per-soc="rangePerSoc"
				:heating="heating"
				@limit-soc-updated="limitSocUpdated"
			/>
			<LimitEnergySelect
				v-else
				class="flex-grow-1 text-end"
				:limit-energy="limitEnergy"
				:soc-per-kwh="socPerKwh"
				:charged-energy="chargedEnergy"
				:capacity="capacity"
				@limit-energy-updated="limitEnergyUpdated"
			/>
		</div>
	</div>
</template>

<script lang="ts">
import collector from "@/mixins/collector.ts";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import LabelAndValue from "../Helper/LabelAndValue.vue";
import Title from "./Title.vue";
import Soc from "./Soc.vue";
import Status from "./Status.vue";
import ChargingPlan from "../ChargingPlans/ChargingPlan.vue";
import LimitSocSelect from "./LimitSocSelect.vue";
import LimitEnergySelect from "./LimitEnergySelect.vue";
import { distanceUnit } from "@/units.ts";
import { defineComponent, type PropType } from "vue";
import {
	CHARGE_MODE,
	type BATTERY_MODE,
	type Forecast,
	type VehicleStatus,
	type Vehicle,
} from "@/types/evcc";
import type { PlanStrategy } from "@/components/ChargingPlans/types";
import BatteryBoostButton from "../Loadpoints/BatteryBoostButton.vue";
import type ChargingPlanModal from "../ChargingPlans/ChargingPlanModal.vue";

export default defineComponent({
	name: "Vehicle",
	components: {
		VehicleTitle: Title,
		VehicleSoc: Soc,
		VehicleStatus: Status,
		LabelAndValue,
		ChargingPlan,
		LimitSocSelect,
		LimitEnergySelect,
		BatteryBoostButton,
	},
	mixins: [collector, formatter],
	props: {
		chargedEnergy: { type: Number, default: 0 },
		charging: Boolean,
		vehicleClimaterActive: Boolean,
		vehicleWelcomeActive: Boolean,
		connected: Boolean,
		currency: String,
		effectiveLimitSoc: Number,
		effectivePlanSoc: Number,
		effectivePlanTime: String,
		effectivePlanStrategy: Object as PropType<PlanStrategy>,
		batteryBoost: Boolean,
		batteryBoostActive: Boolean,
		batteryBoostAvailable: Boolean,
		batteryBoostLimit: { type: Number, default: 100 },
		batterySoc: Number,
		batteryMode: String as PropType<BATTERY_MODE>,
		enabled: Boolean,
		heating: Boolean,
		continuous: Boolean,
		id: [String, Number],
		integratedDevice: Boolean,
		limitEnergy: Number,
		mode: String as PropType<CHARGE_MODE>,
		chargerStatusReason: String,
		phaseAction: String,
		phaseRemainingInterpolated: Number,
		forecast: Object as PropType<Forecast>,
		planActive: Boolean,
		planEnergy: Number,
		planProjectedStart: String,
		planProjectedEnd: String,
		planTime: String,
		planTimeUnreachable: Boolean,
		planOverrun: Number,
		pvAction: String,
		pvRemainingInterpolated: Number,
		sessionSolarPercentage: Number,
		smartCostActive: Boolean,
		smartCostNextStart: String,
		smartCostLimit: Number,
		smartCostType: String,
		smartFeedInPriorityActive: Boolean,
		smartFeedInPriorityNextStart: String,
		smartFeedInPriorityLimit: Number,
		socBasedCharging: Boolean,
		socBasedPlanning: Boolean,
		tariffCo2: Number,
		tariffGrid: Number,
		tariffFeedIn: Number,
		vehicle: Object as PropType<Vehicle>,
		vehicleDetectionActive: Boolean,
		vehicleName: String,
		vehicleRange: { type: Number, default: 0 },
		vehicles: Array,
		vehicleSoc: { type: Number, default: 0 },
		vehicleLimitSoc: Number,
		vehicleNotReachable: Boolean,
		minSocNotReached: Boolean,
		capacity: Number,
		range: Number,
		rangePerSoc: Number,
		socPerKwh: { type: Number, required: true },
	},
	emits: [
		"limit-soc-updated",
		"limit-energy-updated",
		"change-vehicle",
		"remove-vehicle",
		"open-loadpoint-settings",
		"batteryboost-updated",
		"open-modal",
	],
	data() {
		return {
			displayLimitSoc: this.effectiveLimitSoc,
			statusOverride: undefined as VehicleStatus | undefined,
			chargingPlanModal: this.$refs["chargingPlanModal"] as
				| InstanceType<typeof ChargingPlanModal>
				| undefined,
		};
	},
	computed: {
		title() {
			return this.vehicle?.title || "";
		},
		icon() {
			return this.vehicle?.icon || "";
		},
		minSoc() {
			return this.vehicle?.minSoc || 0;
		},
		vehicleSocProps() {
			return this.collectProps(Soc);
		},
		vehicleStatus() {
			return { ...this.collectProps(Status), statusOverride: this.statusOverride };
		},
		vehicleTitleProps() {
			return this.collectProps(Title);
		},
		chargingPlan() {
			return this.collectProps(ChargingPlan);
		},
		showBoostButton(): boolean {
			return this.connected && this.batteryBoostAvailable && this.batteryBoostLimit < 100;
		},
		batteryBoostButtonProps() {
			return this.collectProps(BatteryBoostButton);
		},
		formattedSoc() {
			if (!this.vehicleSoc) {
				return "--";
			}
			if (this.heating) {
				return this.fmtTemperature(this.vehicleSoc);
			}
			return this.fmtPercentage(this.vehicleSoc);
		},
		vehicleSocTitle() {
			if (this.heating) {
				return this.$t("main.vehicle.temp");
			}
			return this.$t("main.vehicle.vehicleSoc");
		},
		rangeUnit() {
			return distanceUnit();
		},
		chargedSoc() {
			const value = this.socPerKwh * (this.chargedEnergy / 1e3);
			return value > 1 ? `+${this.fmtPercentage(value)}` : null;
		},
		chargingPlanDisabled() {
			return this.mode && [CHARGE_MODE.OFF, CHARGE_MODE.NOW].includes(this.mode);
		},
		smartCostDisabled() {
			return this.chargingPlanDisabled;
		},
		smartFeedInPriorityDisabled() {
			return this.chargingPlanDisabled;
		},
	},
	watch: {
		effectiveLimitSoc() {
			this.displayLimitSoc = this.effectiveLimitSoc;
		},
	},
	methods: {
		limitSocDrag(limitSoc: number) {
			this.displayLimitSoc = limitSoc;
		},
		limitSocUpdated(limitSoc: number) {
			this.displayLimitSoc = limitSoc;
			this.$emit("limit-soc-updated", limitSoc);
		},
		limitEnergyUpdated(limitEnergy: number) {
			this.$emit("limit-energy-updated", limitEnergy);
		},
		changeVehicle(name: string) {
			this.$emit("change-vehicle", name);
		},
		removeVehicle() {
			this.$emit("remove-vehicle");
		},
		fmtEnergy(value: number) {
			return this.fmtWh(value, value == 0 ? POWER_UNIT.KW : POWER_UNIT.AUTO);
		},
		openPlanModal(openArrivalTab = false) {
			this.$emit("open-modal", openArrivalTab);
		},
		handleBoostStatus(status: VehicleStatus) {
			this.statusOverride = status;
		},
	},
});
</script>

<style scoped>
.details > div {
	flex-grow: 1;
	flex-basis: 0;
}
</style>
