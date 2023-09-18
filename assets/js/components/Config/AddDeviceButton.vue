<template>
	<li class="root">
		<button class="d-flex align-items-center justify-content-center p-3" @click="$emit('add')">
			<shopicon-regular-plus size="m"></shopicon-regular-plus>
		</button>
	</li>
</template>

<script>
import "@h2d2/shopicons/es/regular/plus";

export default {
	name: "AddDeviceButton",
	emits: ["add"],
};
</script>

<style scoped>
.root {
	border-radius: 1rem;
	border: 1px solid var(--evcc-gray);
	color: var(--evcc-gray);
	padding: 0;
	display: block;
	list-style-type: none;
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
