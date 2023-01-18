<template>
	<div if="plan" class="small text-muted">
		<div>{{ $t("main.targetCharge.planDuration") }}: {{ planDuration }}</div>
		<div>
			{{ $t("main.targetCharge.planPeriodLabel") }}:
			<span v-if="planStart && planEnd">
				{{
					$t("main.targetCharge.planPeriodValue", { start: planStart, end: planEnd })
				}}</span
			>
			<span v-else>{{ $t("main.targetCharge.planUnknown") }}:</span>
		</div>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";

export default {
	name: "TargetChargePlanMinimal",
	mixins: [formatter],
	props: {
		duration: Number,
		plan: Array,
		unit: String,
		power: Number,
	},
	computed: {
		planDuration() {
			return this.fmtShortDuration(this.duration, true);
		},
		lastSlot() {
			if (this.plan?.length) {
				return this.plan[this.plan?.length - 1];
			}
			return null;
		},
		firstSlot() {
			if (this.plan?.length) {
				return this.plan[0];
			}
			return null;
		},
		planStart() {
			if (this.firstSlot) {
				return this.weekdayTime(new Date(this.firstSlot.start));
			}
			return null;
		},
		planEnd() {
			if (this.lastSlot) {
				return this.weekdayTime(new Date(this.lastSlot.end));
			}
			return null;
		},
		durationHours() {
			return this.duration / 3.6e12;
		},
	},
};
</script>

<style scoped></style>
