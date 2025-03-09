<template>
	<div>
		<button
			type="button"
			class="btn gap-2 btn-sm"
			:class="{ active, withLabel: !!label, hideLabelOnMobile }"
			:disabled="disabled"
			@click="$emit('click', value)"
		>
			<slot></slot>
			<span
				v-if="label"
				class="text-nowrap text-truncate"
				:class="{ 'd-none d-md-inline': hideLabelOnMobile }"
			>
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
		hideLabelOnMobile: Boolean,
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
@media (max-width: 768px) {
	.btn.hideLabelOnMobile {
		width: 32px;
		padding: 0;
	}
}
</style>
