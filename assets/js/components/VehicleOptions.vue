<template>
	<div>
		<div
			role="button"
			tabindex="0"
			data-bs-toggle="dropdown"
			data-bs-target="#navbarNavAltMarkup"
			aria-controls="navbarNavAltMarkup"
			aria-expanded="false"
			aria-label="Toggle navigation"
		>
			<slot />
		</div>
		<ul class="dropdown-menu dropdown-menu-start">
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
