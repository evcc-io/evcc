<template>
	<div>
		<div class="mb-3">
			{{ socTitle || $t("main.vehicle.fallbackName") }}
		</div>
		<VehicleSoc v-bind="vehicleSoc" @target-soc-updated="targetSocUpdated" />
		<VehicleSubline v-bind="vehicleSubline" class="my-1" />
	</div>
</template>

<script>
import collector from "../mixins/collector";

import VehicleSoc from "./VehicleSoc";
import VehicleSubline from "./VehicleSubline";

export default {
	name: "Vehicle",
	components: { VehicleSoc, VehicleSubline },
	props: {
		connected: Boolean,
		hasVehicle: Boolean,
		socCharge: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		socTitle: String,
		timerActive: Boolean,
		timerSet: Boolean,
		targetTime: String,
		targetSoC: Number,
	},
	computed: {
		vehicleSoc: function () {
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
	},
	mixins: [collector],
};
</script>
