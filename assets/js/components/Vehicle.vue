<template>
	<div class="vehicle p-4">
		<div class="d-flex justify-content-between mb-2 align-items-center">
			<h4 class="d-flex align-items-center m-0">
				<shopicon-regular-car3 size="m" class="me-2"></shopicon-regular-car3>
				{{ vehicleTitle || $t("main.vehicle.fallbackName") }}
			</h4>
			<button class="btn btn-link text-white p-0">
				<shopicon-filled-options size="s"></shopicon-filled-options>
			</button>
		</div>
		<VehicleStatus
			v-bind="vehicleStatus"
			class="mb-2"
			@target-time-updated="setTargetTime"
			@target-time-removed="removeTargetTime"
		/>
		<VehicleSoc v-bind="vehicleSocProps" class="mb-4" @target-soc-updated="targetSocUpdated" />
		<div class="d-flex flex-wrap justify-content-between">
			<LabelAndValue
				class="flex-grow-1 text-start flex-basis-0"
				:label="$t('main.vehicle.vehicleSoC')"
				:value="`${vehicleSoC || '--'} %`"
				:extraValue="vehicleRange ? `${vehicleRange} km` : null"
			/>
			<LabelAndValue
				class="flex-grow-1 text-end text-sm-center flex-basis-0"
				:label="$t('main.vehicle.targetSoC')"
				:value="`${displayTargetSoC} %`"
			/>
			<TargetCharge
				class="flex-grow-1 text-sm-end target-charge flex-basis-0"
				v-bind="targetCharge"
				@target-time-updated="setTargetTime"
				@target-time-removed="removeTargetTime"
			/>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/filled/options";
import "@h2d2/shopicons/es/regular/car3";

import collector from "../mixins/collector";
import LabelAndValue from "./LabelAndValue.vue";
import VehicleSoc from "./VehicleSoc.vue";
import VehicleStatus from "./VehicleStatus.vue";
import TargetCharge from "./TargetCharge.vue";

export default {
	name: "Vehicle",
	components: { VehicleSoc, VehicleStatus, LabelAndValue, TargetCharge },
	mixins: [collector],
	props: {
		id: Number,
		connected: Boolean,
		vehiclePresent: Boolean,
		vehicleSoC: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		vehicleRange: Number,
		vehicleTitle: String,
		targetTimeActive: Boolean,
		targetTimeHourSuggestion: Number,
		targetTime: String,
		targetSoC: Number,
		phaseAction: String,
		phaseRemainingInterpolated: Number,
		pvAction: String,
		pvRemainingInterpolated: Number,
	},
	data() {
		return {
			displayTargetSoC: this.targetSoC,
		};
	},
	computed: {
		vehicleSocProps: function () {
			return this.collectProps(VehicleSoc);
		},
		vehicleStatus: function () {
			return this.collectProps(VehicleStatus);
		},
		targetCharge: function () {
			return this.collectProps(TargetCharge);
		},
	},
	methods: {
		targetSocUpdated: function (targetSoC) {
			this.displayTargetSoC = targetSoC;
			this.$emit("target-soc-updated", targetSoC);
		},
		setTargetTime: function (targetTime) {
			this.$emit("target-time-updated", targetTime);
		},
		removeTargetTime: function () {
			this.$emit("target-time-removed");
		},
	},
};
</script>

<style scoped>
.vehicle {
	background-color: var(--bs-gray-dark);
	border-radius: 20px;
	color: var(--bs-white);
}
.flex-basis-0 {
	flex-basis: 0;
}
.target-charge {
	min-width: 100%;
}
@media (min-width: 576px) {
	.target-charge {
		min-width: auto;
	}
}
</style>
