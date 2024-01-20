<template>
	<div class="vehicle pt-4">
		<VehicleTitle
			v-if="!integratedDevice"
			v-bind="vehicleTitleProps"
			@change-vehicle="changeVehicle"
			@remove-vehicle="removeVehicle"
		/>
		<VehicleStatus v-bind="vehicleStatus" class="mb-2" />
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
				:extraValue="range ? `${fmtNumber(range, 0)} ${rangeUnit}` : null"
				align="start"
			/>
			<LabelAndValue
				v-else
				class="flex-grow-1"
				:label="$t('main.loadpoint.charged')"
				:value="fmtEnergy(chargedEnergy)"
				:extraValue="chargedSoc"
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

<script>
import collector from "../mixins/collector";
import formatter from "../mixins/formatter";
import LabelAndValue from "./LabelAndValue.vue";
import VehicleTitle from "./VehicleTitle.vue";
import VehicleSoc from "./VehicleSoc.vue";
import VehicleStatus from "./VehicleStatus.vue";
import ChargingPlan from "./ChargingPlan.vue";
import LimitSocSelect from "./LimitSocSelect.vue";
import LimitEnergySelect from "./LimitEnergySelect.vue";
import { distanceUnit, distanceValue } from "../units";

export default {
	name: "Vehicle",
	components: {
		VehicleTitle,
		VehicleSoc,
		VehicleStatus,
		LabelAndValue,
		ChargingPlan,
		LimitSocSelect,
		LimitEnergySelect,
	},
	mixins: [collector, formatter],
	props: {
		chargedEnergy: Number,
		charging: Boolean,
		vehicleClimaterActive: Boolean,
		connected: Boolean,
		currency: String,
		effectiveLimitSoc: Number,
		effectivePlanSoc: Number,
		effectivePlanTime: String,
		enabled: Boolean,
		guardAction: String,
		guardRemainingInterpolated: Number,
		heating: Boolean,
		id: [String, Number],
		integratedDevice: Boolean,
		limitEnergy: Number,
		mode: String,
		phaseAction: String,
		phaseRemainingInterpolated: Number,
		planActive: Boolean,
		planEnergy: Number,
		planProjectedStart: String,
		planTime: String,
		planOverrun: Boolean,
		pvAction: String,
		pvRemainingInterpolated: Number,
		smartCostActive: Boolean,
		smartCostLimit: Number,
		smartCostType: String,
		socBasedCharging: Boolean,
		socBasedPlanning: Boolean,
		tariffCo2: Number,
		tariffGrid: Number,
		vehicle: Object,
		vehicleDetectionActive: Boolean,
		vehicleName: String,
		vehicleRange: Number,
		vehicles: Array,
		vehicleSoc: Number,
		vehicleTargetSoc: Number,
	},
	emits: ["limit-soc-updated", "limit-energy-updated", "change-vehicle", "remove-vehicle"],
	data() {
		return {
			displayLimitSoc: this.effectiveLimitSoc,
		};
	},
	computed: {
		title: function () {
			return this.vehicle?.title || "";
		},
		capacity: function () {
			return this.vehicle?.capacity || 0;
		},
		icon: function () {
			return this.vehicle?.icon || "";
		},
		minSoc: function () {
			return this.vehicle?.minSoc || 0;
		},
		vehicleSocProps: function () {
			return this.collectProps(VehicleSoc);
		},
		vehicleStatus: function () {
			return this.collectProps(VehicleStatus);
		},
		vehicleTitleProps: function () {
			return this.collectProps(VehicleTitle);
		},
		chargingPlan: function () {
			return this.collectProps(ChargingPlan);
		},
		formattedSoc: function () {
			if (!this.vehicleSoc) {
				return "--";
			}
			if (this.heating) {
				return this.fmtTemperature(this.vehicleSoc);
			}
			return `${Math.round(this.vehicleSoc)}%`;
		},
		vehicleSocTitle: function () {
			if (this.heating) {
				return this.$t("main.vehicle.temp");
			}
			return this.$t("main.vehicle.vehicleSoc");
		},
		range: function () {
			return distanceValue(this.vehicleRange);
		},
		rangeUnit: function () {
			return distanceUnit();
		},
		rangePerSoc: function () {
			if (this.vehicleSoc > 10 && this.range) {
				return Math.round((this.range / this.vehicleSoc) * 1e2) / 1e2;
			}
			return null;
		},
		socPerKwh: function () {
			if (this.capacity > 0) {
				return 100 / this.capacity;
			}
			return null;
		},
		chargedSoc: function () {
			const value = this.socPerKwh * (this.chargedEnergy / 1e3);
			return value > 1 ? `+${Math.round(value)}%` : null;
		},
		chargingPlanDisabled: function () {
			if (!this.connected) {
				return true;
			}
			if (["off", "now"].includes(this.mode)) {
				return true;
			}
			return false;
		},
	},
	watch: {
		effectiveLimitSoc: function () {
			this.displayLimitSoc = this.effectiveLimitSoc;
		},
	},
	methods: {
		limitSocDrag: function (limitSoc) {
			this.displayLimitSoc = limitSoc;
		},
		limitSocUpdated: function (limitSoc) {
			this.displayLimitSoc = limitSoc;
			this.$emit("limit-soc-updated", limitSoc);
		},
		limitEnergyUpdated: function (limitEnergy) {
			this.$emit("limit-energy-updated", limitEnergy);
		},
		changeVehicle(name) {
			this.$emit("change-vehicle", name);
		},
		removeVehicle() {
			this.$emit("remove-vehicle");
		},
		fmtEnergy(value) {
			const inKw = value == 0 || value >= 1000;
			return this.fmtKWh(value, inKw);
		},
		openPlanModal() {
			this.$refs.chargingPlan.openPlanModal();
		},
	},
};
</script>

<style scoped>
.details > div {
	flex-grow: 1;
	flex-basis: 0;
}
</style>
