<template>
	<label class="position-relative d-block">
		<select :value="selected" class="custom-select" @change="change">
			<option
				v-for="{ name, value, count } in options"
				:key="value"
				:value="value"
				:disabled="count === 0"
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
	right: 0;
	position: absolute;
	opacity: 0;
	-webkit-appearance: menulist-button;
}
</style>
