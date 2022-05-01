<template>
	<div class="root">
		<div class="mb-2 label">{{ label }}</div>
		<slot>
			<h3 class="value m-0">
				{{ value }}
				<span v-if="extraValue" class="extraValue d-block d-sm-inline text-nowrap">
					{{ extraValue }}
				</span>
			</h3>
		</slot>
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
.root {
	margin-bottom: 1rem;
}
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
