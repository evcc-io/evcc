<template>
	<div class="vehicle pt-4">
		<VehicleTitle
			v-bind="vehicleTitleProps"
			@change-vehicle="changeVehicle"
			@remove-vehicle="removeVehicle"
		/>
		<VehicleStatus v-if="!parked" v-bind="vehicleStatus" class="mb-2" />
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
				:label="$t('main.vehicle.vehicleSoC')"
				:value="vehicleSoC ? `${vehicleSoC}%` : '--'"
				:extraValue="vehicleRange ? `${vehicleRange} km` : null"
				align="start"
			/>
			<LabelAndValue
				v-else
				class="flex-grow-1"
				:label="$t('main.loadpoint.charged')"
				:value="fmtEnergy(chargedEnergy)"
				:extraValue="chargedSoC"
				align="start"
			/>
			<TargetCharge
				v-if="socBasedCharging"
				class="flex-grow-1 text-center target-charge"
				v-bind="targetCharge"
				:disabled="targetChargeDisabled"
				@target-time-updated="setTargetTime"
				@target-time-removed="removeTargetTime"
			/>
			<TargetSoCSelect
				v-if="socBasedCharging"
				class="flex-grow-1 text-end"
				:target-soc="displayTargetSoC"
				:range-per-soc="rangePerSoC"
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
		<div v-if="$hiddenFeatures" class="d-flex justify-content-start">
			<small>vor 5 Stunden</small>
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
import TargetCharge from "./TargetCharge.vue";
import TargetSoCSelect from "./TargetSoCSelect.vue";
import TargetEnergySelect from "./TargetEnergySelect.vue";

export default {
	name: "Vehicle",
	components: {
		VehicleTitle,
		VehicleSoc,
		VehicleStatus,
		LabelAndValue,
		TargetCharge,
		TargetSoCSelect,
		TargetEnergySelect,
	},
	mixins: [collector, formatter],
	props: {
		id: [String, Number],
		connected: Boolean,
		vehiclePresent: Boolean,
		vehicleSoC: Number,
		vehicleTargetSoC: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		vehicleDetectionActive: Boolean,
		vehicleRange: Number,
		vehicleTitle: String,
		vehicleCapacity: Number,
		socBasedCharging: Boolean,
		targetTimeActive: Boolean,
		targetTime: String,
		targetTimeProjectedStart: String,
		targetSoC: Number,
		targetEnergy: Number,
		chargedEnergy: Number,
		mode: String,
		phaseAction: String,
		phaseRemainingInterpolated: Number,
		pvAction: String,
		pvRemainingInterpolated: Number,
		parked: Boolean,
		vehicles: Array,
	},
	emits: [
		"target-time-removed",
		"target-time-updated",
		"target-soc-updated",
		"target-energy-updated",
		"change-vehicle",
		"remove-vehicle",
	],
	data() {
		return {
			displayTargetSoC: this.targetSoC,
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
		targetCharge: function () {
			return this.collectProps(TargetCharge);
		},
		rangePerSoC: function () {
			if (this.vehicleSoC > 10 && this.vehicleRange) {
				return this.vehicleRange / this.vehicleSoC;
			}
			return null;
		},
		socPerKwh: function () {
			if (this.vehicleCapacity > 0) {
				return 100 / this.vehicleCapacity;
			}
			return null;
		},
		chargedSoC: function () {
			const value = this.socPerKwh * (this.chargedEnergy / 1e3);
			return value > 1 ? `+${Math.round(value)}%` : null;
		},
		targetChargeDisabled: function () {
			return !this.connected || !["pv", "minpv"].includes(this.mode);
		},
	},
	watch: {
		targetSoC: function () {
			this.displayTargetSoC = this.targetSoC;
		},
	},
	methods: {
		targetSocDrag: function (targetSoC) {
			this.displayTargetSoC = targetSoC;
		},
		targetSocUpdated: function (targetSoC) {
			this.displayTargetSoC = targetSoC;
			this.$emit("target-soc-updated", targetSoC);
		},
		targetEnergyUpdated: function (targetEnergy) {
			this.$emit("target-energy-updated", targetEnergy);
		},
		setTargetTime: function (targetTime) {
			this.$emit("target-time-updated", targetTime);
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
