<template>
	<label class="position-relative d-block" :for="id">
		<select :id="id" :value="selected" class="custom-select" @change="change">
			<option
				v-for="{ name, value, count, disabled } in options"
				:key="value"
				:value="value"
				:disabled="count === 0 || disabled"
			>
				{{ text(name, count) }}
			</option>
		</select>
		<slot></slot>
	</label>
</template>

<script>
export default {
	name: "CustomSelect",
	props: {
		options: { type: Array },
		selected: { type: String },
		id: { type: String },
	},
	emits: ["change"],
	methods: {
		text(name, count) {
			if (count === undefined) {
				return name;
			}
			return `${name} (${count})`;
		},
		change(event) {
			this.$emit("change", event);
		},
	},
};
</script>
<style scoped>
.custom-select {
	left: 0;
	top: 0;
	bottom: 0;
	width: 100%;
	position: absolute;
	opacity: 0;
	-webkit-appearance: menulist-button;
}
</style>
