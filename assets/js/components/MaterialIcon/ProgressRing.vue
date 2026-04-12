<template>
	<svg
		class="progressRing"
		:style="{ '--duration': `${duration}ms`, ...svgStyle }"
		viewBox="0 0 20 20"
	>
		<circle class="track" cx="10" cy="10" r="6" fill="none" stroke="currentColor" />
		<circle class="bar" cx="10" cy="10" r="6" fill="none" stroke="currentColor" />
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "ProgressRing",
	mixins: [icon],
	props: {
		duration: { type: Number, required: true },
	},
});
</script>

<style scoped>
.progressRing {
	background: transparent;
	transform: rotate(-90deg);
}
.progressRing .track {
	fill: none;
	stroke: currentColor;
	stroke-width: 2.5;
	opacity: 0.25;
}
.progressRing .bar {
	fill: none;
	stroke: currentColor;
	stroke-width: 2.5;
	stroke-dasharray: 40;
	stroke-dashoffset: 40;
	animation: progressRing var(--duration, 10s) cubic-bezier(0.4, 0.1, 0.6, 0.9) forwards;
}
@keyframes progressRing {
	from {
		stroke-dashoffset: 40;
	}
	to {
		stroke-dashoffset: 0;
	}
}
</style>
