<template>
	<div>
		<div class="d-flex align-items-start">
			<div class="mb-3">
				{{ vehicleTitle || $t("main.vehicle.fallbackName") }}
			</div>
			<div v-if="$hiddenFeatures" class="dropdown ms-2" style="margin-top: -0.1rem">
				<button
					:id="`vehicle_dropdown_${id}`"
					class="btn btn-sm btn-link text-muted"
					type="button"
					data-bs-toggle="dropdown"
					aria-expanded="false"
				>
					<fa-icon icon="fa-solid fa-right-left"></fa-icon>
				</button>
				<ul class="dropdown-menu" :aria-labelledby="`vehicle_dropdown_${id}`">
					<h6 class="dropdown-header">Fahrzeug wechseln</h6>
					<button class="dropdown-item" type="button" @click="removeVehicle">
						Gastfahrzeug
					</button>
					<button class="dropdown-item" type="button" @click="changeVehicle">
						blaues Model 3
					</button>
					<button class="dropdown-item" type="button" @click="changeVehicle">
						Lenas e-Niro
					</button>
				</ul>
			</div>
		</div>
		<VehicleSoc v-bind="vehicleSocProps" @target-soc-updated="targetSocUpdated" />
		<VehicleSubline
			v-bind="vehicleSubline"
			class="my-1"
			@target-time-updated="setTargetTime"
			@target-time-removed="removeTargetTime"
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
		targetTimeActive: Boolean,
		targetTimeHourSuggestion: Number,
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
		setTargetTime: function (targetTime) {
			this.$emit("target-time-updated", targetTime);
		},
		removeTargetTime: function () {
			this.$emit("target-time-removed");
		},
		removeVehicle: function () {
			this.$emit("vehicle-removed");
		},
		changeVehicle: function () {
			window.alert("Fahrzeug wechseln");
		},
	},
};
</script>
