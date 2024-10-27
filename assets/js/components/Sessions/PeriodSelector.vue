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

<script>
import SelectGroup from "../SelectGroup.vue";

export default {
	name: "PeriodSelector",
	components: {
		SelectGroup,
	},
	props: {
		period: { type: String, required: true },
		periodOptions: { type: Array, required: true },
	},
	emits: ["update:period"],
	methods: {
		changePeriod(newPeriod) {
			this.$emit("update:period", newPeriod);
		},
	},
};
</script>
