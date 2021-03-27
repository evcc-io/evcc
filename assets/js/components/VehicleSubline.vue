<template>
	<div class="subline text-secondary d-flex justify-content-between align-items-center">
		<div>
			<div v-if="minSoCActive">
				<fa-icon class="text-muted mr-1" icon="exclamation-circle"></fa-icon>
				Mindestladung bis {{ minSoC }}%
			</div>
		</div>
		<button
			class="btn btn-link btn-sm pr-0"
			:class="{ 'text-dark': timerSet, 'text-muted': !timerSet }"
			@click="selectTargetTime"
		>
			{{ targetTimeLabel() }}<fa-icon class="ml-1" icon="clock"></fa-icon>
		</button>
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
			if (this.targetChargeEnabled) {
				const targetDate = Date.parse(this.targetTime);
				if (this.timerActive) {
					return `Lädt ${this.fmtRelativeTime(targetDate)} bis ${this.targetSoC}%`;
				} else {
					return `Geplant bis ${this.fmtAbsoluteDate(targetDate)} bis ${this.targetSoC}%`;
				}
			}
			return "Zielzeit festlegen";
		},
		selectTargetTime: function () {
			window.alert("Bis wann soll geladen werden?");
		},
	},
	mixins: [formatter],
};
</script>
<style scoped>
.subline {
	font-size: 0.875rem;
}
</style>
