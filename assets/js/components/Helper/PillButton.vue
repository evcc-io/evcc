<template>
	<button
		type="button"
		class="root btn btn-sm rounded-pill d-flex align-items-center gap-2 px-3"
		@click="$emit('click')"
	>
		<slot />
	</button>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
	name: "PillButton",
	emits: ["click"],
});
</script>

<style scoped>
.root {
	border: 1px solid var(--evcc-gray-50);
	color: inherit;
	transition: border-color var(--evcc-transition-fast) linear;
}
.root:hover,
.root:focus-within {
	border-color: var(--evcc-default-text);
	color: var(--evcc-default-text);
}
</style>
