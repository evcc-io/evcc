<template>
	<div class="phases d-flex justify-content-between" :class="rootClass">
		<div v-for="num in [1, 2, 3]" :key="num" :class="`phase phase--${num} me-1`">
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
		chargeCurrents: { type: Array },
		phasesActive: { type: Number },
		minCurrent: { type: Number },
		maxCurrent: { type: Number },
	},
	computed: {
		rootClass() {
			return this.onlyFirstPhase ? "phases--1p" : "phases-3p";
		},
		onlyFirstPhase() {
			if (this.chargeCurrents) {
				const [l1, l2, l3] = this.chargeCurrents;
				return l1 && !l2 && !l3;
			}
			return this.phasesActive === 1;
		},
	},
	methods: {
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
	width: 73px;
}
.phase {
	background-color: var(--bs-gray-bright);
	height: 4px;
	flex-grow: 1;
	position: relative;
	border-radius: 1px;
	overflow: hidden;
	flex-basis: 100%;
	opacity: 1;
	transition-property: flex-basis, margin, opacity;
	transition-duration: var(--evcc-transition-slow);
	transition-timing-function: ease-in;
}
html.dark .phase {
	background-color: var(--bs-gray-bright);
}

.phase.inactive {
	flex-basis: 0;
	margin-right: 0 !important;
	opacity: 0;
}
.target,
.real {
	position: absolute;
	left: 0;
	top: 0;
	bottom: 0;
	transition-property: width, opacity;
	transition-duration: var(--evcc-transition-slow);
	transition-timing-function: ease-in;
	opacity: 1;
}
.target {
	background-color: var(--evcc-green);
}
.real {
	background-color: var(--evcc-dark-green);
}
</style>
