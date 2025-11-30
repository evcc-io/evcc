<template>
	<SelectGroup
		id="sessionsPeriodSmall"
		class="w-100 d-flex d-lg-none"
		:options="periodOptions"
		:modelValue="period"
		@update:model-value="changePeriod"
	/>
	<SelectGroup
		id="sessionsPeriod"
		class="w-100 d-none d-lg-flex"
		:options="periodOptions"
		large
		:modelValue="period"
		@update:model-value="changePeriod"
	/>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import SelectGroup from "../Helper/SelectGroup.vue";
import type { SelectOption } from "@/types/evcc";
import type { PERIODS } from "./types";

export default defineComponent({
	name: "PeriodSelector",
	components: {
		SelectGroup,
	},
	props: {
		period: { type: String, required: true },
		periodOptions: { type: Array as PropType<SelectOption<PERIODS>[]>, required: true },
	},
	emits: ["update:period"],
	methods: {
		changePeriod(newPeriod: PERIODS) {
			this.$emit("update:period", newPeriod);
		},
	},
});
</script>
