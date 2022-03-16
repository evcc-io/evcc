<template>
	<div
		class="phases d-flex flex-column justify-content-between"
		:class="`active-phases-${activePhases}`"
	>
		<div v-for="num in [1, 2, 3]" :key="num" class="phase" :class="{ inactive: inactive(num) }">
			<div class="target" :style="{ width: `${targetWidth()}%` }"></div>
			<div class="real" :style="{ width: `${realWidth(num)}%` }"></div>
		</div>
	</div>
</template>

<script>
export default {
	name: "Phases",
	props: {
		chargeCurrent: { type: Number },
		chargeCurrents: { type: Object },
		activePhases: { type: Number },
		minCurrent: { type: Number },
		maxCurrent: { type: Number },
		phaseAction: { type: String },
		phaseRemaining: { type: Number },
	},
	computed: {
		phaseTimerVisible() {
			if (this.phaseTimerActive && !this.pvTimerActive) {
				return true;
			}
			if (this.phaseTimerActive && this.pvTimerActive) {
				return this.phaseRemainingInterpolated < this.pvRemainingInterpolated; // only show next timer
			}
			return false;
		},
		pvTimerVisible() {
			if (this.pvTimerActive && !this.phaseTimerActive) {
				return true;
			}
			if (this.pvTimerActive && this.phaseTimerActive) {
				return this.pvRemainingInterpolated < this.phaseRemainingInterpolated; // only show next timer
			}
			return false;
		},
	},
	methods: {
		inactive(num) {
			return num > this.activePhases;
		},
		targetWidth() {
			let current = Math.min(Math.max(this.minCurrent, this.chargeCurrent), this.maxCurrent);
			return (100 / this.maxCurrent) * current;
		},
		realWidth(num) {
			if (this.chargeCurrents) {
				const current = this.chargeCurrents[num - 1] || 0;
				return (100 / this.maxCurrent) * current;
			}
			return this.targetWidth();
		},
	},
};
</script>

<style scoped>
.phases {
	height: 11px;
}
.phase {
	background-color: var(--bs-gray-200);
	height: 3px;
	width: 100%;
	position: relative;
}
.target,
.real {
	position: absolute;
	left: 0;
	top: 0;
	bottom: 0;
	transition-property: width, opacity;
	transition-duration: 0.75s;
	transition-timing-function: ease-in;
	opacity: 1;
}
.target {
	background-color: var(--evcc-green);
}
.real {
	background-color: var(--evcc-dark-green);
}
.phase.inactive .target,
.phase.inactive .real {
	opacity: 0;
}
</style>
