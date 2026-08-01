<template>
	<button
		type="button"
		class="root d-inline-flex align-items-center justify-content-center rounded-circle"
		:title="title"
		:aria-label="title"
		@click="$emit('click')"
	>
		<shopicon-regular-plus size="s"></shopicon-regular-plus>
	</button>
</template>

<script>
import "@h2d2/shopicons/es/regular/plus";

export default {
	name: "AddDeviceButton",
	props: {
		title: String,
	},
	emits: ["click"],
};
</script>

<style scoped>
.root {
	width: 2rem;
	height: 2rem;
	padding: 0;
	border: none;
	background-color: var(--evcc-dark-green);
	color: var(--evcc-background);
	transition: opacity var(--evcc-transition-fast) linear;
}
.root:hover,
.root:focus-visible {
	opacity: 0.75;
}
</style>
