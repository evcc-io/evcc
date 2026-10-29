<template>
	<div
		v-tooltip="phasesMismatch ? $t('main.loadpoint.phasesMismatch') : undefined"
		class="phases d-flex"
		:class="{ 'phases-warning': phasesMismatch }"
	>
		<div
			v-for="num in [1, 2, 3]"
			:key="num"
			class="phase me-1"
			:class="{
				'phase-inactive': !isPhaseActive(num),
				'phase-hidden': !isPhaseActive(num) && (maxPhases === 1 || num > maxPhases),
			}"
		>
			<div
				v-show="targetWidth() > 0"
				class="target"
				:style="{ width: `${targetWidth()}%` }"
			></div>
			<div class="real" :style="{ width: `${realWidth(num)}%` }"></div>
		</div>
	</div>
</template>

<script lang="ts">
import type { PHASES } from "@/types/evcc";
import { defineComponent, type PropType } from "vue";
const MIN_ACTIVE_CURRENT = 1;

export default defineComponent({
	name: "Phases",
	props: {
		offeredCurrent: { type: Number, default: 0 },
		chargeCurrents: { type: Array as PropType<number[]> },
		phasesActive: { type: Number as PropType<PHASES> },
		phasesConfigured: { type: Number, default: 3 },
		minCurrent: { type: Number, default: 6 },
		maxCurrent: { type: Number, default: 16 },
	},
	computed: {
		chargeCurrentsActive() {
			if (!this.chargeCurrents) return false;
			return this.chargeCurrents.filter((c) => c >= MIN_ACTIVE_CURRENT).length > 0;
		},
		maxPhases() {
			// 0 = automatic 1p3p switching
			return this.phasesConfigured || 3;
		},
		phasesMismatch() {
			return [1, 2, 3].filter((num) => this.isPhaseActive(num)).length > this.maxPhases;
		},
	},
	methods: {
		targetWidth() {
			if (this.offeredCurrent <= 0) return 0;
			const current = Math.min(
				Math.max(this.minCurrent, this.offeredCurrent),
				this.maxCurrent
			);
			return (100 / this.maxCurrent) * current;
		},
		realWidth(num: number) {
			if (this.chargeCurrents) {
				const current = this.chargeCurrents[num - 1] || 0;
				return (100 / this.maxCurrent) * current;
			}
			return this.targetWidth();
		},
		isPhaseActive(num: number) {
			if (this.chargeCurrentsActive && this.chargeCurrents) {
				const current = this.chargeCurrents[num - 1];
				return current !== undefined && current >= MIN_ACTIVE_CURRENT;
			}
			// unknown active phases: assume all
			return num <= (this.phasesActive || this.maxPhases);
		},
	},
});
</script>

<style scoped>
.phases {
	width: 73px;
}
.phase {
	background-color: var(--evcc-gray-15);
	height: 4px;
	flex: 1 1 0;
	position: relative;
	border-radius: 2px;
	overflow: hidden;
	opacity: 1;
	transition-property: flex-basis, flex-grow, margin, opacity;
	transition-duration: var(--evcc-transition-slow);
	transition-timing-function: ease-in;
}
html.dark .phase {
	background-color: var(--bs-gray-bright);
}
.phase-inactive {
	flex: 0 0 4px;
}
.phase-inactive .target,
.phase-inactive .real {
	opacity: 0;
}
.phase-hidden {
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
	background-color: color-mix(in srgb, var(--evcc-dark-green) 40%, white);
}
.real {
	background-color: var(--evcc-dark-green);
}
.phases-warning .target {
	background-color: color-mix(in srgb, var(--bs-warning) 40%, white);
}
.phases-warning .real {
	background-color: var(--bs-warning);
}
</style>
