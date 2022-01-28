<template>
	<div>
		<div class="mb-2 label">{{ label }}</div>
		<h3 class="value">
			{{ value }}<span v-if="extraValue" class="extraValue"> / {{ extraValue }}</span>
		</h3>
	</div>
</template>

<script>
export default {
	name: "LabelAndValue",
	props: {
		label: String,
		value: String,
		extraValue: String,
	},
};
</script>
<style scoped>
.label {
	text-transform: uppercase;
	color: var(--bs-gray-medium);
	font-size: 14px;
}
.value {
	font-size: 18px;
}
.extraValue {
	color: var(--bs-gray-medium);
	font-size: 14px;
}
</style>
