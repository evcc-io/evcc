<template>
	<div class="mb-4 row">
		<label
			:for="id"
			:class="`col-sm-${leftWidth} col-form-label d-sm-flex align-items-sm-baseline`"
		>
			<div class="w-100">
				<span class="label">{{ label }}</span>
				<small v-if="optional" class="d-sm-block ms-2 ms-sm-0">optional</small>
			</div>
		</label>
		<div :class="`col-sm-${rightWidth}`">
			<slot />
		</div>
	</div>
</template>

<script>
export default {
	name: "FormRow",
	props: {
		id: String,
		label: String,
		optional: Boolean,
		smallValue: Boolean,
	},
	computed: {
		leftWidth() {
			return this.smallValue ? "6" : "4";
		},
		rightWidth() {
			return this.smallValue ? "6" : "8";
		},
	},
};
</script>
<style scoped>
.label {
	-webkit-hyphens: auto;
	hyphens: auto;
	max-width: 100%;
}
</style>
