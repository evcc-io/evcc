<template>
	<div class="container p-0">
		<div
			v-for="(vehicle, index) in vehicles"
			:key="index"
			class="flex-grow-1 mx-4 mx-md-5 mb-4"
		>
			<Vehicle v-bind="vehicle" :id="index" class="vehicle" />
		</div>
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
		return { vehicles: [{ name: "A" }, { name: "B" }] };
	},
	methods: {},
};
</script>
<style scoped>
.vehicle {
	border: 4px solid white;
}
</style>
