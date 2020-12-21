<template>
	<div>
		<label :for="id"
			><span class="font-weight-bold">{{ title }}</span> ({{ value }}{{ unit }})</label
		>
		<input
			type="range"
			class="custom-range"
			:id="id"
			:min="min"
			:max="max"
			:step="step"
			v-model="value"
		/>
	</div>
</template>

<script>
export default {
	name: "Range",
	props: {
		id: String,
		title: String,
		unit: String,
		val: [String, Number],
		min: {
			default: 0,
			type: Number,
		},
		max: {
			default: 100,
			type: Number,
		},
		step: {
			default: 5,
			type: Number,
		},
	},
	data: function () {
		return {
			value: null,
		};
	},
	watch: {
		value: function (val, old) {
			if (old !== null) {
				this.$emit("updated", val);
			}
		},
	},
	mounted: function () {
		this.value = this.val;
	},
};
</script>
