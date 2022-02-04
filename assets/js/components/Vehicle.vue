<template>
	<div class="vehicle p-4">
		<div class="d-flex justify-content-between mb-2 align-items-center">
			<h4 class="d-flex align-items-center m-0">
				<shopicon-regular-car3 size="m" class="me-2"></shopicon-regular-car3>
				{{ vehicleTitle || $t("main.vehicle.fallbackName") }}
			</h4>
			<button class="btn btn-link text-white p-0">
				<shopicon-filled-options size="s"></shopicon-filled-options>
			</button>
		</div>
		<VehicleStatus
			v-bind="vehicleStatus"
			class="mb-2"
			@target-time-updated="setTargetTime"
			@target-time-removed="removeTargetTime"
		/>
		<VehicleSoc v-bind="vehicleSocProps" class="mb-4" @target-soc-updated="targetSocUpdated" />
		<div class="d-flex">
			<LabelAndValue
				class="flex-grow-1"
				:label="$t('main.vehicle.vehicleSoC')"
				:value="`${vehicleSoC} %`"
				:extraValue="vehicleRange ? `${vehicleRange} km` : null"
			/>
			<LabelAndValue
				class="flex-grow-1"
				:label="$t('main.vehicle.targetSoC')"
				:value="`${displayTargetSoC} %`"
			/>
		</div>
		<hr class="divider my-3" />
		<TargetCharge
			v-bind="targetCharge"
			@target-time-updated="setTargetTime"
			@target-time-removed="removeTargetTime"
		/>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/filled/options";
import "@h2d2/shopicons/es/regular/car3";

import LabelAndValue from "./LabelAndValue";
import collector from "../mixins/collector";

import VehicleSoc from "./VehicleSoc";
import VehicleStatus from "./VehicleStatus";
import TargetCharge from "./TargetCharge";

export default {
	name: "Vehicle",
	components: { VehicleSoc, VehicleStatus, LabelAndValue, TargetCharge },
	mixins: [collector],
	props: {
		id: Number,
		connected: Boolean,
		vehiclePresent: Boolean,
		vehicleSoC: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		vehicleRange: Number,
		vehicleTitle: String,
		targetTimeActive: Boolean,
		targetTimeHourSuggestion: Number,
		targetTime: String,
		targetSoC: Number,
		phaseAction: String,
		phaseRemainingInterpolated: Number,
		pvAction: String,
		pvRemainingInterpolated: Number,
	},
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
		targetCharge: function () {
			return this.collectProps(TargetCharge);
		},
	},
	methods: {
		targetSocUpdated: function (targetSoC) {
			this.displayTargetSoC = targetSoC;
			this.$emit("target-soc-updated", targetSoC);
		},
		setTargetTime: function (targetTime) {
			this.$emit("target-time-updated", targetTime);
		},
		removeTargetTime: function () {
			this.$emit("target-time-removed");
		},
	},
};
</script>

<style scoped>
.vehicle {
	background-color: var(--bs-gray-dark);
	border-radius: 20px;
	color: var(--bs-white);
}
.divider {
	border: none;
	border-top: 1px solid var(--bs-gray-medium);
}
</style>
