<template>
	<button
		type="button"
		class="root d-inline-flex align-items-center"
		:title="title"
		:aria-label="title"
		@click="$emit('click')"
	>
		<AddCircle />
	</button>
</template>

<script>
import AddCircle from "../MaterialIcon/AddCircle.vue";

export default {
	name: "AddDeviceButton",
	components: { AddCircle },
	props: {
		title: String,
	},
	emits: ["click"],
};
</script>

<style scoped>
.root {
	padding: 0;
	border: none;
	background: none;
	color: var(--evcc-dark-green);
	transition: opacity var(--evcc-transition-fast) linear;
}
.root:hover,
.root:focus-visible {
	opacity: 0.75;
}
</style>
