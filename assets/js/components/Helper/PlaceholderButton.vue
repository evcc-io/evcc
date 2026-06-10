<template>
	<button
		type="button"
		class="root d-flex align-items-center justify-content-center"
		:disabled="disabled"
		@click="$emit('click')"
	>
		<slot />
	</button>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "PlaceholderButton",
	props: {
		disabled: Boolean,
	},
	emits: ["click"],
});
</script>

<style scoped>
.root {
	border-radius: 1rem;
	border: 1px solid var(--evcc-gray-50);
	padding: 1.5rem;
	background: none;
	width: 100%;
	color: var(--evcc-gray);
	transition:
		border-color var(--evcc-transition-fast) linear,
		color var(--evcc-transition-fast) linear;
}
.root:hover:not(:disabled),
.root:focus-within:not(:disabled) {
	border-color: var(--evcc-default-text);
	color: var(--evcc-default-text);
}
.root:disabled {
	opacity: 0.5;
	cursor: not-allowed;
}
</style>
