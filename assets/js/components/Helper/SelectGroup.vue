<template>
	<div class="mode-group border d-inline-flex" :class="{ large, transparent }" role="group">
		<button
			v-for="(option, i) in options"
			:id="i === 0 ? id : undefined"
			:key="option.value"
			type="button"
			class="btn btn-sm flex-grow-1 flex-shrink-1"
			:class="{ active: option.value === modelValue, 'btn--equal': equalWidth }"
			:disabled="option.disabled"
			:data-testid="`${id}-${option.value}`"
			tabindex="0"
			@click="$emit('update:modelValue', option.value)"
		>
			{{ option.name }}
		</button>
	</div>
</template>

<script lang="ts">
import type { SelectOption } from "@/types/evcc";
import { defineComponent, type PropType } from "vue";

export default defineComponent({
	name: "SelectGroup",
	props: {
		id: String,
		options: Array as PropType<SelectOption<string>[]>,
		modelValue: [Number, String, Boolean],
		equalWidth: Boolean,
		large: Boolean,
		transparent: Boolean,
	},
	emits: ["update:modelValue"],
});
</script>

<style scoped>
.mode-group {
	border: 2px solid var(--evcc-default-text);
	border-radius: 17px;
	padding: 4px;
}

.mode-group:not(.transparent) {
	background: var(--evcc-background);
}

.btn {
	white-space: nowrap;
	border-radius: 12px;
	padding: 0.1em 0.8em;
	color: var(--evcc-default-text);
	border: none;
	overflow-x: hidden;
	text-overflow: ellipsis;
}
.btn--equal {
	flex-basis: 0;
}
.btn:hover {
	color: var(--evcc-gray);
}
.btn:focus {
	outline: var(--bs-focus-ring-width) solid var(--bs-focus-ring-color);
	outline-width: var(--bs-focus-ring-width);
}
.btn.active {
	color: var(--evcc-background);
	background: var(--evcc-default-text);
}
.modal-group.large {
	height: 32px;
}
.large .btn {
	height: 32px;
	border-radius: 16px;
}
</style>
