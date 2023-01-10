<template>
	<div>
		<div
			:id="dropdownId"
			role="button"
			tabindex="0"
			data-bs-toggle="dropdown"
			aria-expanded="false"
		>
			<slot />
		</div>
		<ul class="dropdown-menu dropdown-menu-start" :aria-labelledby="dropdownId">
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
import Dropdown from "bootstrap/js/dist/dropdown";

export default {
	name: "VehicleOptions",
	props: {
		id: [String, Number],
		vehicles: Array,
		isUnknown: Boolean,
	},
	emits: ["change-vehicle", "remove-vehicle"],
	computed: {
		dropdownId() {
			return `vehicleOptionsDropdown${this.id}`;
		},
	},
	mounted() {
		this.dropdown = new Dropdown(document.getElementById(this.dropdownId));
	},
	unmounted() {
		this.dropdown?.dispose();
	},
	methods: {
		changeVehicle(index) {
			this.$emit("change-vehicle", index + 1);
		},
		removeVehicle() {
			this.$emit("remove-vehicle");
		},
	},
};
</script>

<style></style>
