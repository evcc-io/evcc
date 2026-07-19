<template>
	<span class="successCheck">
		<shopicon-regular-checkmark></shopicon-regular-checkmark>
	</span>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/checkmark";
import { defineComponent } from "vue";

export default defineComponent({
	name: "SuccessCheck",
});
</script>

<style scoped>
.successCheck {
	display: flex;
	align-items: center;
	justify-content: center;
	width: 2.5rem;
	height: 2.5rem;
	border-radius: 50%;
	background: var(--evcc-green);
	color: var(--evcc-darkest-green);
	animation: successCheckIn var(--evcc-transition-fast, 250ms) ease-in both;
}
@keyframes successCheckIn {
	from {
		transform: scale(0.5);
		opacity: 0;
	}
	to {
		transform: scale(1);
		opacity: 1;
	}
}
</style>
