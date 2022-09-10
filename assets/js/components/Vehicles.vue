<template>
	<div>
		<div class="container px-4 mb-3 mb-sm-4 d-flex justify-content-start align-items-center">
			<h2 class="m-0">{{ $t("main.vehicles") }}</h2>
			<button
				class="btn btn-link d-flex evcc-default-text p-2 ms-1 refresh"
				:class="{
					'refresh--in-progress': refreshing,
				}"
				:disabled="refreshing"
				@click="refresh"
			>
				<shopicon-regular-refresh class="refresh-icon"></shopicon-regular-refresh>
			</button>
		</div>

		<div class="container vehicles px-0 mb-5" :class="`vehicles-${vehicles.length}`">
			<Vehicle
				v-for="(vehicle, index) in vehicles"
				v-bind="vehicle"
				:id="`vehicle_${index}`"
				:key="index"
				class="vehicle"
				parked
			/>
		</div>
	</div>
</template>

<script>
import Vehicle from "./Vehicle.vue";
import "@h2d2/shopicons/es/regular/refresh";

export default {
	name: "Vehicles",
	components: { Vehicle },

	data() {
		return {
			refreshing: false,
			// TODO: mock data for development
			vehicles: [
				{
					vehiclePresent: true,
					vehicleTitle: "Ford Mustang Mach-E",
					vehicleSoC: 46,
					vehicleRange: 182,
					targetSoC: 100,
				},
				{
					vehiclePresent: true,
					vehicleTitle: "Renault Twingo Electric",
					vehicleSoC: 77,
					vehicleRange: 98,
					targetSoC: 90,
				},
				{
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
	methods: {
		refresh() {
			this.refreshing = true;
			// TODO: insert real implementation here
			window.setTimeout(() => {
				this.refreshing = false;
			}, 5000);
		},
	},
};
</script>
<style scoped>
.vehicles {
	display: grid;
	grid-gap: 2rem;
	grid-template-columns: repeat(auto-fit, minmax(310px, 1fr));
}
.vehicle {
	border: 4px solid white;
}
.refresh--in-progress {
	animation: rotation 1s infinite cubic-bezier(0.37, 0, 0.63, 1);
}
.refresh-icon {
	transform: translateY(-2px);
}
@keyframes rotation {
	from {
		transform: rotate(0deg);
	}
	to {
		transform: rotate(360deg);
	}
}
</style>
