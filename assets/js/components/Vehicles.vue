<template>
	<div class="container p-0 vehicles px-2 px-lg-5 mb-5">
		<Vehicle
			v-for="(vehicle, index) in vehicles"
			v-bind="vehicle"
			:id="index"
			:key="index"
			class="vehicle"
		/>
	</div>
</template>

<script>
import Vehicle from "./Vehicle.vue";
import collector from "../mixins/collector";

export default {
	name: "Vehicles",
	components: { Vehicle },
	mixins: [collector],

	data() {
		return {
			vehicles: [
				{
					connected: true,
					vehiclePresent: true,
					vehicleTitle: "Ford Mustang Mach-E",
					vehicleSoC: 46,
					vehicleRange: 182,
					targetSoC: 100,
				},
				{
					connected: true,
					vehiclePresent: true,
					vehicleTitle: "Renault Twingo Electric",
					vehicleSoC: 77,
					vehicleRange: 98,
					targetSoC: 90,
				},
				{
					connected: true,
					vehiclePresent: true,
					vehicleTitle: "Blauer VW ID.4",
					vehicleSoC: 16,
					vehicleRange: 52,
					minSoC: 35,
					targetSoC: 60,
				},
			],
		};
	},
	methods: {},
};
</script>
<style scoped>
.vehicles {
	display: grid;
	grid-gap: 4rem;
	grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
}
.vehicle {
	border: 4px solid white;
}
</style>
