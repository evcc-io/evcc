<template>
	<div>
		<div
			:id="dropdownId"
			role="button"
			tabindex="0"
			data-bs-toggle="dropdown"
			aria-expanded="false"
			data-testid="change-vehicle"
		>
			<slot />
		</div>
		<ul class="dropdown-menu dropdown-menu-start" :aria-labelledby="dropdownId">
			<li>
				<h6 class="dropdown-header">{{ $t("main.vehicle.changeVehicle") }}</h6>
			</li>
			<li v-for="vehicle in vehicles" :key="vehicle.name">
				<button type="button" class="dropdown-item" @click="changeVehicle(vehicle.name)">
					{{ vehicle.title }}
				</button>
			</li>
			<li>
				<button type="button" class="dropdown-item" @click="removeVehicle()">
					<span v-if="connected">{{ $t("main.vehicle.unknown") }}</span>
					<span v-else>{{ $t("main.vehicle.none") }}</span>
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
		connected: Boolean,
		id: [String, Number],
		vehicles: Array,
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
		changeVehicle(name) {
			this.$emit("change-vehicle", name);
		},
		removeVehicle() {
			this.$emit("remove-vehicle");
		},
	},
};
</script>

<style></style>
