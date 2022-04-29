<template>
	<div class="vehicle p-4 pb-3">
		<VehicleTitle v-bind="vehicleTitleProps" />
		<VehicleStatus v-if="!parked" v-bind="vehicleStatus" class="mb-2" />
		<VehicleSoc
			v-bind="vehicleSocProps"
			class="mt-2 mb-4"
			@target-soc-updated="targetSocUpdated"
			@target-soc-drag="targetSocDrag"
		/>
		<div v-if="vehiclePresent">
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
					:extraValue="estimatedTargetRange"
				/>
				<TargetCharge
					class="flex-grow-1 text-sm-end target-charge flex-basis-0"
					v-bind="targetCharge"
					:disabled="targetChargeDisabled"
					@target-time-updated="setTargetTime"
					@target-time-removed="removeTargetTime"
				/>
			</div>
			<div v-if="$hiddenFeatures" class="d-flex justify-content-start">
				<small>vor 5 Stunden</small>
			</div>
		</div>
	</div>
</template>

<script>
import collector from "../mixins/collector";
import LabelAndValue from "./LabelAndValue.vue";
import VehicleTitle from "./VehicleTitle.vue";
import VehicleSoc from "./VehicleSoc.vue";
import VehicleStatus from "./VehicleStatus.vue";
import TargetCharge from "./TargetCharge.vue";

export default {
	name: "Vehicle",
	components: { VehicleTitle, VehicleSoc, VehicleStatus, LabelAndValue, TargetCharge },
	mixins: [collector],
	props: {
		id: [String, Number],
		connected: Boolean,
		vehiclePresent: Boolean,
		vehicleSoC: Number,
		enabled: Boolean,
		charging: Boolean,
		minSoC: Number,
		vehicleRange: Number,
		vehicleTitle: String,
		targetTimeActive: Boolean,
		targetTime: String,
		targetTimeProjectedStart: String,
		targetSoC: Number,
		mode: String,
		phaseAction: String,
		phaseRemainingInterpolated: Number,
		pvAction: String,
		pvRemainingInterpolated: Number,
		parked: Boolean,
	},
	emits: ["target-time-removed", "target-time-updated", "target-soc-updated"],
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
		vehicleTitleProps: function () {
			return this.collectProps(VehicleTitle);
		},
		targetCharge: function () {
			return this.collectProps(TargetCharge);
		},
		estimatedTargetRange: function () {
			if (this.vehicleSoC > 10 && this.vehicleRange && this.displayTargetSoC) {
				return (
					Math.round(this.displayTargetSoC * (this.vehicleRange / this.vehicleSoC)) +
					" km"
				);
			}
			return null;
		},
		targetChargeDisabled: function () {
			return !this.connected || !["pv", "minpv"].includes(this.mode);
		},
	},
	watch: {
		targetSoC: function () {
			this.displayTargetSoC = this.targetSoC;
		},
	},
	methods: {
		targetSocDrag: function (targetSoC) {
			this.displayTargetSoC = targetSoC;
		},
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
	border-radius: 1rem;
	color: var(--bs-white);
}
.car-icon {
	width: 1.75rem;
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
