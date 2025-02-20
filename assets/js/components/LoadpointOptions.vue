<template>
	<label
		class="position-relative d-block"
		:for="dropdownId"
		role="button"
		data-testid="change-loadpoint"
	>
		<select :id="dropdownId" :value="selected" class="custom-select" @change="change">
			<option
				v-for="name in loadpoints"
				:key="name"
				:value="name"
				:selected="name === selected"
			>
				{{ name }}
			</option>
			<hr />
			<option value="" :selected="!selected">
				{{ $t(`main.loadpoint.none`) }}
			</option>
		</select>
		<slot></slot>
	</label>
</template>

<script>
export default {
	name: "LoadpointOptions",
	props: {
		id: [String, Number],
		loadpoints: Array,
		selected: String,
	},
	emits: ["change-loadpoint", "remove-loadpoint"],
	computed: {
		dropdownId() {
			return `loadpointOptionsDropdown${this.id}`;
		},
	},
	methods: {
		change(event) {
			const name = event.target.value;
			console.log(name);

			if (name) {
				this.$emit("change-loadpoint", name);
			} else {
				this.$emit("remove-loadpoint");
			}
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
	cursor: pointer;
	position: absolute;
	opacity: 0;
	-webkit-appearance: menulist-button;
}
</style>
