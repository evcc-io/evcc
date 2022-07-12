<template>
	<div>
		<button
			type="button"
			data-bs-toggle="dropdown"
			data-bs-target="#navbarNavAltMarkup"
			aria-controls="navbarNavAltMarkup"
			aria-expanded="false"
			aria-label="Toggle navigation"
			class="btn btn-sm btn-outline-secondary position-relative border-0 menu-button"
		>
			<shopicon-filled-options size="s"></shopicon-filled-options>
		</button>
		<ul class="dropdown-menu dropdown-menu-end">
			<li>
				<h6 class="dropdown-header">{{ $t("main.vehicle.changeVehicle") }}</h6>
			</li>
			<li v-for="vehicle in vehicles" :key="vehicle">
				<button type="button" class="dropdown-item" @click="changeVehicle(vehicle.id)">
					{{ vehicle.title }}
				</button>
			</li>
			<li v-if="!isUnknown">
				<button type="button" class="dropdown-item" @click="removeVehicle()">
					{{ $t("main.vehicle.unknown") }}
				</button>
			</li>
		</ul>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/filled/options";

export default {
	name: "VehicleOptions",
	props: {
		vehicles: Array,
		isUnknown: Boolean,
	},
	emits: ["change-vehicle", "remove-vehicle"],
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

<style></style>
