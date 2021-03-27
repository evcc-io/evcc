<template>
	<div>
		<div class="mb-2 pb-1">
			{{ socTitle || "Fahrzeug" }}
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
		socTitle: String,
		connected: Boolean,
		hasVehicle: Boolean,
		socCharge: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		timerActive: Boolean,
		timerSet: Boolean,
		targetTime: String,
		targetSoC: Number,
	},
	data: function () {
		return {
			selectedTargetSoC: null,
		};
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
		targetSocUpdated: function (value) {
			this.$emit("target-soc-updated", value);
		},
	},
	mixins: [collector],
};
</script>
