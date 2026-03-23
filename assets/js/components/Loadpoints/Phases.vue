<template>
	<div :class="`phases d-flex justify-content-between`">
		<div
			v-for="num in [1, 2, 3]"
			:key="num"
			class="phase me-1"
			:class="{ 'phase-inactive': !isPhaseActive(num) }"
		>
			<div class="target" :style="{ width: `${targetWidth()}%` }"></div>
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
		minCurrent: { type: Number, default: 6 },
		maxCurrent: { type: Number, default: 16 },
	},
	computed: {
		chargeCurrentsActive() {
			if (!this.chargeCurrents) return false;
			return this.chargeCurrents.filter((c) => c >= MIN_ACTIVE_CURRENT).length > 0;
		},
	},
	methods: {
		targetWidth() {
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
			return num <= (this.phasesActive || 0);
		},
	},
});
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
.phase-inactive {
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
