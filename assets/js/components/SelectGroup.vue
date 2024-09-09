<template>
	<div class="mode-group border d-inline-flex" role="group">
		<button
			v-for="(option, i) in options"
			:id="i === 0 ? id : null"
			:key="option.value"
			type="button"
			class="btn btn-lg flex-grow-1 flex-shrink-1"
			:class="{ active: option.value === modelValue, 'btn--equal': equalWidth }"
			:disabled="option.disabled"
			@click="$emit('update:modelValue', option.value)"
		>
			{{ option.name }}
		</button>
	</div>
</template>

<script>
export default {
	name: "SelectGroup",
	props: {
		id: String,
		options: Array,
		modelValue: [Number, String],
		equalWidth: Boolean,
	},
	emits: ["update:modelValue"],
};
</script>

<style scoped>
.mode-group {
	border: 2px solid var(--evcc-default-text);
	border-radius: 17px;
	padding: 4px;
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
.btn.active {
	color: var(--evcc-background);
	background: var(--evcc-default-text);
}
.btn-group {
	border-radius: 16px;
}
</style>
