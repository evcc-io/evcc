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
			@target-soc-updated="targetSocUpdated"
			@target-soc-drag="targetSocDrag"
		/>

		<div class="details d-flex flex-wrap justify-content-between">
			<LabelAndValue
				v-if="socBasedCharging"
				class="flex-grow-1"
				:label="$t('main.vehicle.vehicleSoc')"
				:value="vehicleSoc ? `${Math.round(vehicleSoc)}%` : '--'"
				:extraValue="range ? `${Math.round(range)} ${rangeUnit}` : null"
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
				class="flex-grow-1 target-charge"
				v-bind="chargingPlan"
				:disabled="chargingPlanDisabled"
				@target-time-updated="setTargetTime"
				@target-time-removed="removeTargetTime"
				@minsoc-updated="setMinSoc"
			/>
			<TargetSocSelect
				v-if="socBasedCharging"
				class="flex-grow-1 text-end"
				:target-soc="displayTargetSoc"
				:range-per-soc="rangePerSoc"
				@target-soc-updated="targetSocUpdated"
			/>
			<TargetEnergySelect
				v-else
				class="flex-grow-1 text-end"
				:target-energy="targetEnergy"
				:soc-per-kwh="socPerKwh"
				:charged-energy="chargedEnergy"
				:vehicle-capacity="vehicleCapacity"
				@target-energy-updated="targetEnergyUpdated"
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
import TargetSocSelect from "./TargetSocSelect.vue";
import TargetEnergySelect from "./TargetEnergySelect.vue";
import { distanceUnit, distanceValue } from "../units";

export default {
	name: "Vehicle",
	components: {
		VehicleTitle,
		VehicleSoc,
		VehicleStatus,
		LabelAndValue,
		ChargingPlan,
		TargetSocSelect,
		TargetEnergySelect,
	},
	mixins: [collector, formatter],
	props: {
		id: [String, Number],
		connected: Boolean,
		integratedDevice: Boolean,
		vehiclePresent: Boolean,
		vehicleSoc: Number,
		vehicleTargetSoc: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoc: Number,
		vehicleDetectionActive: Boolean,
		vehicleRange: Number,
		vehicleTitle: String,
		vehicleIcon: String,
		vehicleCapacity: Number,
		socBasedCharging: Boolean,
		planActive: Boolean,
		planProjectedStart: String,
		targetTime: String,
		targetSoc: Number,
		targetEnergy: Number,
		chargedEnergy: Number,
		mode: String,
		phaseAction: String,
		phaseRemainingInterpolated: Number,
		pvAction: String,
		pvRemainingInterpolated: Number,
		guardAction: String,
		guardRemainingInterpolated: Number,
		vehicles: Array,
		climaterActive: Boolean,
		smartCostLimit: Number,
		smartCostType: String,
		tariffGrid: Number,
		tariffCo2: Number,
		currency: String,
	},
	emits: [
		"target-time-removed",
		"target-time-updated",
		"target-soc-updated",
		"target-energy-updated",
		"change-vehicle",
		"remove-vehicle",
		"minsoc-updated",
	],
	data() {
		return {
			displayTargetSoc: this.targetSoc,
		};
	},
	computed: {
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
		range: function () {
			return distanceValue(this.vehicleRange);
		},
		rangeUnit: function () {
			return distanceUnit();
		},
		rangePerSoc: function () {
			if (this.vehicleSoc > 10 && this.range) {
				return this.range / this.vehicleSoc;
			}
			return null;
		},
		socPerKwh: function () {
			if (this.vehicleCapacity > 0) {
				return 100 / this.vehicleCapacity;
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
			// enabled for vehicles with Soc
			if (this.socBasedCharging) {
				return false;
			}
			// disabled of no energy target is set (offline or guest vehicles)
			if (!this.targetEnergy) {
				return true;
			}

			return false;
		},
	},
	watch: {
		targetSoc: function () {
			this.displayTargetSoc = this.targetSoc;
		},
	},
	methods: {
		targetSocDrag: function (targetSoc) {
			this.displayTargetSoc = targetSoc;
		},
		targetSocUpdated: function (targetSoc) {
			this.displayTargetSoc = targetSoc;
			this.$emit("target-soc-updated", targetSoc);
		},
		targetEnergyUpdated: function (targetEnergy) {
			this.$emit("target-energy-updated", targetEnergy);
		},
		setTargetTime: function (targetTime) {
			this.$emit("target-time-updated", targetTime);
		},
		setMinSoc: function (minSoc) {
			this.$emit("minsoc-updated", minSoc);
		},
		removeTargetTime: function () {
			this.$emit("target-time-removed");
		},
		changeVehicle(index) {
			this.$emit("change-vehicle", index);
		},
		removeVehicle() {
			this.$emit("remove-vehicle");
		},
		fmtEnergy(value) {
			const inKw = value == 0 || value >= 1000;
			return this.fmtKWh(value, inKw);
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
