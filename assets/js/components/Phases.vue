<template>
	<div
		class="phases d-flex flex-column justify-content-between"
		:class="`active-phases-${activePhases}`"
	>
		<div
			v-for="num in [1, 2, 3]"
			:key="num"
			class="phase"
			:class="{ [`phase-${num}`]: true, 'd-none': inactive(num) }"
		>
			<div class="inner" :style="{ width: `${width(num)}%` }"></div>
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
		width(num) {
			let current = Math.min(Math.max(this.minCurrent, this.chargeCurrent), this.maxCurrent);
			if (this.chargeCurrents) {
				current = this.chargeCurrents[num - 1] || 0;
			}
			return (100 / this.maxCurrent) * current;
		},
	},
};
</script>

<style scoped>
.phases {
	height: 10px;
}
.phase {
	background-color: var(--evcc-green);
}
.active-phases-1 .phase {
	height: 10px;
}
.active-phases-2 .phase {
	height: 4px;
}
.active-phases-3 .phase {
	height: 2px;
}
.inner {
	height: 100%;
	background-color: var(--evcc-dark-green);
}
</style>
