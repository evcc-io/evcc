<template>
	<div class="d-flex justify-content-between mb-3 align-items-center">
		<h4 class="d-flex align-items-center m-0 flex-grow-1 overflow-hidden">
			<shopicon-regular-car3
				v-if="carIcon"
				class="me-2 flex-shrink-0 car-icon"
			></shopicon-regular-car3>
			<shopicon-regular-cablecharge
				v-else
				class="me-2 flex-shrink-0 car-icon"
			></shopicon-regular-cablecharge>
			<span class="flex-grow-1 text-truncate"> {{ name }} </span>
		</h4>
		<VehicleOptions
			v-if="showOptions"
			class="options"
			:vehicles="otherVehicles"
			:is-unknown="isUnknown"
			@change-vehicle="changeVehicle"
			@remove-vehicle="removeVehicle"
		/>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/car3";
import "@h2d2/shopicons/es/regular/cablecharge";

import VehicleOptions from "./VehicleOptions.vue";

export default {
	name: "VehicleTitle",
	components: { VehicleOptions },
	props: {
		vehiclePresent: Boolean,
		vehicleTitle: String,
		parked: Boolean,
		connected: Boolean,
		vehicles: { type: Array, default: () => [] },
	},
	emits: ["change-vehicle", "remove-vehicle"],
	computed: {
		carIcon() {
			return this.connected || this.parked;
		},
		name() {
			if (this.vehiclePresent || this.parked) {
				return this.vehicleTitle || this.$t("main.vehicle.fallbackName");
			}
			if (this.connected) {
				return this.$t("main.vehicle.unknown");
			}
			return this.$t("main.vehicle.none");
		},
		isUnknown() {
			return this.connected && !this.vehiclePresent;
		},
		otherVehicles() {
			return this.vehicles
				.map((v, id) => ({
					id: id,
					title: v,
				}))
				.filter((v) => v.title !== this.vehicleTitle);
		},
		showOptions() {
			return !this.isUnknown || this.vehicles.length;
		},
	},
	methods: {
		changeVehicle(index) {
			this.$emit("change-vehicle", index);
		},
		removeVehicle() {
			this.$emit("remove-vehicle");
		},
	},
};
</script>

<style scoped>
.options {
	margin-right: -0.75rem;
}
</style>
