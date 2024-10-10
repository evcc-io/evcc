<template>
	<div>
		<button type="button" class="btn" :class="{ active }" @click="$emit('click', value)">
			<slot></slot>
		</button>
	</div>
</template>

<script>
export default {
	name: "IconSelectItem",
	emits: ["click"],
	props: {
		value: String,
		active: Boolean,
	},
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
</style>
