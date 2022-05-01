<template>
	<div class="d-flex justify-content-between mb-3 align-items-center">
		<h4 class="d-flex align-items-center m-0 flex-grow-1 overflow-hidden">
			<shopicon-regular-car3
				v-if="vehiclePresent || parked"
				class="me-2 flex-shrink-0 car-icon"
			></shopicon-regular-car3>
			<shopicon-regular-cablecharge
				v-else
				class="me-2 flex-shrink-0 car-icon"
			></shopicon-regular-cablecharge>
			<span class="flex-grow-1 text-truncate"> {{ name }} </span>
		</h4>
		<button v-if="$hiddenFeatures" class="btn btn-link text-white p-0 flex-shrink-0">
			<shopicon-filled-options size="s"></shopicon-filled-options>
		</button>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/filled/options";
import "@h2d2/shopicons/es/regular/car3";
import "@h2d2/shopicons/es/regular/cablecharge";

import collector from "../mixins/collector";

export default {
	name: "VehicleTitle",
	mixins: [collector],
	props: {
		vehiclePresent: Boolean,
		vehicleTitle: String,
		parked: Boolean,
	},
	computed: {
		name() {
			if (this.vehiclePresent || this.parked) {
				return this.vehicleTitle || this.$t("main.vehicle.fallbackName");
			}
			return this.$t("main.vehicle.noVehicle");
		},
	},
};
</script>
