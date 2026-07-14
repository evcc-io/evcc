<template>
	<CustomSelect
		:id="dropdownId"
		:options="options"
		:selected="selected"
		:aria-label="$t('main.vehicle.changeVehicle')"
		data-testid="change-vehicle"
		@change="change"
	>
		<slot></slot>
	</CustomSelect>
</template>

<script lang="ts">
import CustomSelect from "../Helper/CustomSelect.vue";
import type { SelectOption } from "@/types/evcc";
import { defineComponent, type PropType } from "vue";

const SETTINGS = "_settings";
const DIVIDER = { name: "─────", disabled: true };

export default defineComponent({
	name: "VehicleOptionsWithSettings",
	components: { CustomSelect },
	props: {
		connected: Boolean,
		id: [String, Number],
		vehicleOptions: Array as PropType<SelectOption<string>[]>,
		selected: String,
	},
	emits: ["change-vehicle", "remove-vehicle", "open-settings"],
	computed: {
		dropdownId() {
			return `vehicleOptionsDropdown${this.id}`;
		},
		options(): SelectOption<string>[] {
			// incoming vehicleOptions: name = vehicle name, value = title
			const toOption = (o: SelectOption<string>) => ({ name: o.value, value: o.name });
			const selected = this.vehicleOptions?.find(({ name }) => name === this.selected);
			const others = (this.vehicleOptions || []).filter(({ name }) => name !== this.selected);

			const result: SelectOption<string>[] = [];
			if (selected) {
				result.push(toOption(selected));
				result.push({
					name: `→ ${this.$t("main.vehicleSettings.entry")}`,
					value: SETTINGS,
				});
				result.push({ ...DIVIDER, value: "_divider1" });
			}
			result.push(...others.map(toOption));
			if (others.length) {
				result.push({ ...DIVIDER, value: "_divider2" });
			}
			result.push({
				name: this.$t(`main.vehicle.${this.connected ? "unknown" : "none"}`),
				value: "",
			});
			return result;
		},
	},
	methods: {
		change(event: Event) {
			const target = event.target as HTMLSelectElement;
			const name = target.value;
			if (name === SETTINGS) {
				// keep current vehicle displayed
				target.value = this.selected || "";
				this.$emit("open-settings");
			} else if (name) {
				this.$emit("change-vehicle", name);
			} else {
				this.$emit("remove-vehicle");
			}
		},
	},
});
</script>
