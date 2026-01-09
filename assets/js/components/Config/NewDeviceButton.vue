<template>
	<button
		class="root d-flex align-items-center justify-content-center"
		tabindex="0"
		@click="$emit('click')"
	>
		<shopicon-regular-plus class="me-1"></shopicon-regular-plus>
		<span class="text-start">{{ title }}</span>
	</button>
</template>

<script>
import "@h2d2/shopicons/es/regular/plus";

export default {
	name: "NewDeviceButton",
	props: {
		title: String,
	},
	emits: ["click"],
};
</script>

<style scoped>
.root {
	min-height: 9rem;
	border-radius: 1rem;
	border: 1px solid var(--evcc-gray-50);
	padding: 2rem;
	transition: border-color var(--evcc-transition-fast) linear;
	background: none;
	width: 100%;
	color: inherit;
	margin: 0;
}
.root:hover,
.root:focus-within {
	border-color: var(--evcc-default-text);
	color: var(--evcc-default-text);
}
</style>
