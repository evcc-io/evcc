<template>
	<div class="vehicle p-4">
		<div class="d-flex justify-content-between mb-3 align-items-center">
			<h4 class="d-flex align-items-center">
				<shopicon-regular-car3 size="m" class="me-2"></shopicon-regular-car3>
				{{ vehicleTitle || $t("main.vehicle.fallbackName") }}
			</h4>
			<button class="btn btn-link text-white p-0">
				<shopicon-filled-options size="s"></shopicon-filled-options>
			</button>
		</div>
		<VehicleSoc v-bind="vehicleSocProps" @target-soc-updated="targetSocUpdated" />
		<VehicleSubline
			v-bind="vehicleSubline"
			class="my-1"
			@target-time-updated="setTargetTime"
			@target-time-removed="removeTargetTime"
		/>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/filled/options";
import "@h2d2/shopicons/es/regular/car3";

import collector from "../mixins/collector";

import VehicleSoc from "./VehicleSoc";
import VehicleSubline from "./VehicleSubline";

export default {
	name: "Vehicle",
	components: { VehicleSoc, VehicleSubline },
	mixins: [collector],
	props: {
		id: Number,
		connected: Boolean,
		vehiclePresent: Boolean,
		vehicleSoC: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		vehicleTitle: String,
		targetTimeActive: Boolean,
		targetTimeHourSuggestion: Number,
		targetTime: String,
		targetSoC: Number,
	},
	computed: {
		vehicleSocProps: function () {
			return this.collectProps(VehicleSoc);
		},
		vehicleSubline: function () {
			return this.collectProps(VehicleSubline);
		},
	},
	methods: {
		targetSocUpdated: function (targetSoC) {
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
</style>
