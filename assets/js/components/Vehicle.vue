<template>
	<div>
		<div class="mb-3">
			{{ vehicleTitle || $t("main.vehicle.fallbackName") }}
		</div>
		<VehicleSoc v-bind="vehicleSocProps" @target-soc-updated="targetSocUpdated" />
		<VehicleSubline
			v-bind="vehicleSubline"
			class="my-1"
			@target-time-updated="targetTimeUpdated"
		/>
	</div>
</template>

<script>
import collector from "../mixins/collector";

import VehicleSoc from "./VehicleSoc";
import VehicleSubline from "./VehicleSubline";

export default {
	name: "Vehicle",
	components: { VehicleSoc, VehicleSubline },
	mixins: [collector],
	props: {
		id: Number,
		connected: Boolean,
		vehiclePresent: Boolean,
		vehicleSoC: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		vehicleTitle: String,
		timerActive: Boolean,
		timerSet: Boolean,
		targetTime: String,
		targetSoC: Number,
	},
	computed: {
		vehicleSocProps: function () {
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
		targetTimeUpdated: function (targetTime) {
			this.$emit("target-time-updated", targetTime);
		},
	},
};
</script>
