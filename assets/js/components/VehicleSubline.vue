<template>
	<div class="d-flex justify-content-between align-items-center">
		<small class="text-secondary">
			<span v-if="minSoCActive">
				<fa-icon class="text-muted me-1" icon="exclamation-circle"></fa-icon>
				{{ $t("main.vehicleSubline.mincharge", { soc: minSoC }) }}
			</span>
		</small>
		<small
			v-if="targetChargeEnabled"
			:class="{
				invisible: !targetSoC,
				'text-primary': timerActive,
				'text-secondary': !timerActive,
			}"
		>
			{{ targetTimeLabel() }}
			<fa-icon class="ms-1" icon="clock"></fa-icon>
		</small>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";

export default {
	name: "VehicleSubline",
	props: {
		socCharge: Number,
		minSoC: Number,
		timerActive: Boolean,
		timerSet: Boolean,
		targetTime: String,
		targetSoC: Number,
	},
	computed: {
		minSoCActive: function () {
			return this.minSoC > 0 && this.socCharge < this.minSoC;
		},
		targetChargeEnabled: function () {
			return this.targetTime && this.timerSet;
		},
	},
	methods: {
		// not computed because it needs to update over time
		targetTimeLabel: function () {
			const targetDate = new Date(this.targetTime);
			return `bis ${this.fmtAbsoluteDate(targetDate)} Uhr`;
		},
	},
	mixins: [formatter],
};
</script>
