<template>
	<label class="root position-relative d-block" :for="id" role="button">
		<select :id="id" :value="selected" class="custom-select" tabindex="0" @change="change">
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

<script lang="ts">
import type { SelectOption } from "../../types/evcc";
import { defineComponent, type PropType } from "vue";

export default defineComponent({
	name: "CustomSelect",
	props: {
		options: { type: Array as PropType<SelectOption<number | string>[]> },
		selected: { type: [String, Number] },
		id: { type: String },
	},
	emits: ["change"],
	methods: {
		text(name: string, count?: number) {
			if (count === undefined) {
				return name;
			}
			return `${name} (${count})`;
		},
		change(event: Event) {
			this.$emit("change", event);
		},
	},
});
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
.root {
	border-radius: var(--bs-border-radius);
}
.root:focus-within {
	outline: var(--bs-focus-ring-width) solid var(--bs-focus-ring-color);
	outline-width: var(--bs-focus-ring-width);
	outline-offset: var(--bs-focus-ring-width);
}
</style>
