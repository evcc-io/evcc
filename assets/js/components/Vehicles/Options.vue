<template>
	<label
		class="position-relative d-block"
		:for="dropdownId"
		role="button"
		data-testid="change-vehicle"
	>
		<select :id="dropdownId" :value="selected" class="custom-select" @change="change">
			<option
				v-for="{ name, title } in vehicles"
				:key="name"
				:value="name"
				:selected="name === selected"
			>
				{{ title }}
			</option>
			<hr />
			<option value="" :selected="!selected">
				{{ $t(`main.vehicle.${connected ? "unknown" : "none"}`) }}
			</option>
		</select>
		<slot></slot>
	</label>
</template>

<script>
export default {
	name: "VehicleOptions",
	props: {
		connected: Boolean,
		id: [String, Number],
		vehicles: Array,
		selected: String,
	},
	emits: ["change-vehicle", "remove-vehicle"],
	computed: {
		dropdownId() {
			return `vehicleOptionsDropdown${this.id}`;
		},
	},
	methods: {
		change(event) {
			const name = event.target.value;
			if (name) {
				this.$emit("change-vehicle", name);
			} else {
				this.$emit("remove-vehicle");
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
