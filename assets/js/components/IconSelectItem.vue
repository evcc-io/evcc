<template>
	<div>
		<button
			type="button"
			class="btn gap-2 btn-sm"
			:class="{ active, withLabel: !!label }"
			:disabled="disabled"
			@click="$emit('click', value)"
		>
			<slot></slot>
			<span v-if="label" class="text-nowrap text-truncate">
				{{ label }}
			</span>
		</button>
	</div>
</template>

<script>
export default {
	name: "IconSelectItem",
	props: {
		value: String,
		active: Boolean,
		label: String,
		disabled: Boolean,
	},
	emits: ["click"],
};
</script>

<style scoped>
.btn {
	width: 32px;
	height: 32px;
	border-radius: 2rem;
	color: var(--evcc-default-text);
	padding: 0;
	border: none;
	display: flex;
	align-items: center;
	justify-content: center;
}
.btn:hover {
	color: var(--evcc-gray);
}
.btn.active {
	color: var(--evcc-background);
	background: var(--evcc-default-text);
}
.btn.withLabel {
	width: auto;
	padding: 0 1rem;
}
</style>
