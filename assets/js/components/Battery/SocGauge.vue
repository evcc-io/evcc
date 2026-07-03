<template>
	<svg :width="size" :height="size" :viewBox="`0 0 ${size} ${size}`" class="flex-shrink-0">
		<circle
			:cx="c"
			:cy="c"
			:r="r"
			fill="none"
			stroke="var(--bs-border-color)"
			:stroke-width="stroke"
		/>
		<circle
			:cx="c"
			:cy="c"
			:r="r"
			fill="none"
			:stroke="color"
			:stroke-width="stroke"
			stroke-linecap="round"
			:stroke-dasharray="circumference"
			:stroke-dashoffset="offset"
			:transform="`rotate(-90 ${c} ${c})`"
			class="progress"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "SocGauge",
	props: {
		soc: { type: Number, default: 0 },
		color: { type: String, default: "" },
		size: { type: Number, default: 38 },
	},
	computed: {
		stroke(): number {
			return Math.max(4, Math.round(this.size * 0.12));
		},
		c(): number {
			return this.size / 2;
		},
		r(): number {
			return (this.size - this.stroke) / 2;
		},
		circumference(): number {
			return 2 * Math.PI * this.r;
		},
		offset(): number {
			return this.circumference * (1 - Math.max(0, Math.min(100, this.soc)) / 100);
		},
	},
});
</script>

<style scoped>
.progress {
	transition: stroke-dashoffset var(--evcc-transition-medium) ease;
}
</style>
