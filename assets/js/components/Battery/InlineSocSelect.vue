<template>
	<CustomSelect
		:id="id"
		:options="options"
		:selected="selected"
		inline
		@change="$emit('change', $event)"
	>
		<span class="text-decoration-underline fw-bold" :class="{ 'text-nowrap': nowrap }">
			{{ label }}
		</span>
	</CustomSelect>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import type { SelectOption } from "@/types/evcc";
import CustomSelect from "../Helper/CustomSelect.vue";

// Inline soc picker: an underlined, clickable value that opens a CustomSelect. Used for the
// priority / buffer / buffer-start values embedded in the usage sentences.
export default defineComponent({
	name: "InlineSocSelect",
	components: { CustomSelect },
	props: {
		id: { type: String, required: true },
		options: { type: Array as PropType<SelectOption<number>[]>, default: () => [] },
		selected: { type: Number, default: 0 },
		label: { type: String, default: "" },
		nowrap: Boolean, // keep short values (e.g. "80 %") on one line; off for long phrases
	},
	emits: ["change"],
});
</script>
