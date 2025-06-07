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
			@open-minsoc-settings="openMinSocSettings"
			@open-plan-modal="openPlanModal"
		/>
		<VehicleSoc
			v-bind="vehicleSocProps"
			class="mt-2 mb-4"
			@limit-soc-updated="limitSocUpdated"
			@limit-soc-drag="limitSocDrag"
			@plan-clicked="openPlanModal"
		/>
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
import collector from "@/mixins/collector";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import LabelAndValue from "../Helper/LabelAndValue.vue";
import Title from "./Title.vue";
import Soc from "./Soc.vue";
import Status from "./Status.vue";
import ChargingPlan from "../ChargingPlans/ChargingPlan.vue";
import LimitSocSelect from "./LimitSocSelect.vue";
import LimitEnergySelect from "./LimitEnergySelect.vue";
import { distanceUnit, distanceValue } from "@/units";
import { defineComponent, type PropType } from "vue";
import { CHARGE_MODE, type Forecast, type Vehicle } from "@/types/evcc";

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
		batteryBoostActive: Boolean,
		enabled: Boolean,
		heating: Boolean,
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
		planPrecondition: Number,
		planOverrun: Number,
		pvAction: String,
		pvRemainingInterpolated: Number,
		sessionSolarPercentage: Number,
		smartCostActive: Boolean,
		smartCostNextStart: String,
		smartCostLimit: Number,
		smartCostType: String,
		socBasedCharging: Boolean,
		socBasedPlanning: Boolean,
		tariffCo2: Number,
		tariffGrid: Number,
		vehicle: Object as PropType<Vehicle>,
		vehicleDetectionActive: Boolean,
		vehicleName: String,
		vehicleRange: Number,
		vehicles: Array,
		vehicleSoc: { type: Number, default: 0 },
		vehicleLimitSoc: Number,
		vehicleNotReachable: Boolean,
	},
	emits: [
		"limit-soc-updated",
		"limit-energy-updated",
		"change-vehicle",
		"remove-vehicle",
		"open-loadpoint-settings",
	],
	data() {
		return {
			displayLimitSoc: this.effectiveLimitSoc,
		};
	},
	computed: {
		title() {
			return this.vehicle?.title || "";
		},
		capacity() {
			return this.vehicle?.capacity || 0;
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
			return this.collectProps(Status);
		},
		vehicleTitleProps() {
			return this.collectProps(Title);
		},
		chargingPlan() {
			return this.collectProps(ChargingPlan);
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
		range() {
			return distanceValue(this.vehicleRange);
		},
		rangeUnit() {
			return distanceUnit();
		},
		rangePerSoc() {
			if (this.vehicleSoc > 10 && this.range) {
				return Math.round((this.range / this.vehicleSoc) * 1e2) / 1e2;
			}
			return undefined;
		},
		socPerKwh() {
			if (this.capacity > 0) {
				return 100 / this.capacity;
			}
			return 0;
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
		openPlanModal() {
			(
				this.$refs["chargingPlan"] as InstanceType<typeof ChargingPlan> | undefined
			)?.openPlanModal();
		},
		openMinSocSettings() {
			(
				this.$refs["chargingPlan"] as InstanceType<typeof ChargingPlan> | undefined
			)?.openPlanModal(true);
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
