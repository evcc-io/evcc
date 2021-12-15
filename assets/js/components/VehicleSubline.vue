<template>
	<div class="d-flex justify-content-between align-items-center">
		<small class="text-secondary">
			<span v-if="minSoCActive">
				<fa-icon class="text-muted me-1" icon="exclamation-circle"></fa-icon>
				{{ $t("main.vehicleSubline.mincharge", { soc: minSoC }) }}
			</span>
		</small>
		<TargetCharge v-bind="targetCharge" @target-time-updated="targetTimeUpdated" />
	</div>
</template>

<script>
import collector from "../mixins/collector";
import TargetCharge from "./TargetCharge.vue";

export default {
	name: "VehicleSubline",
	components: { TargetCharge },
	props: {
		id: Number,
		vehicleSoC: Number,
		minSoC: Number,
		timerActive: Boolean,
		timerSet: Boolean,
		targetTime: String,
		targetSoC: Number,
	},
	computed: {
		minSoCActive: function () {
			return this.minSoC > 0 && this.vehicleSoC < this.minSoC;
		},
		targetCharge: function () {
			return this.collectProps(TargetCharge);
		},
	},
	mixins: [collector],
	methods: {
		targetTimeUpdated: function (targetTime) {
			this.$emit("target-time-updated", targetTime);
		},
	},
};
</script>
