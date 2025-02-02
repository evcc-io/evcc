<template>
	<li class="root">
		<button class="d-flex align-items-center justify-content-center p-3" @click="$emit('add')">
			<shopicon-regular-plus class="me-1"></shopicon-regular-plus>
			{{ title }}
		</button>
	</li>
</template>

<script>
import "@h2d2/shopicons/es/regular/plus";

export default {
	name: "AddDeviceButton",
	props: {
		title: String,
	},
	emits: ["add"],
};
</script>

<style scoped>
.root {
	padding: 0;
	display: block;
	list-style-type: none;
	min-height: 9rem;
	border-radius: 1rem;
	border: 1px solid var(--evcc-gray-50);
	padding: 1rem 1rem 0.5rem;
	transition: border-color var(--evcc-transition-fast) linear;
}
.root:hover {
	border-color: var(--evcc-default-text);
	color: var(--evcc-default-text);
}
button {
	border-radius: 1rem;
	background: none;
	border: none;
	height: 100%;
	width: 100%;
	color: inherit;
}
</style>
