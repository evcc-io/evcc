<template>
	<SelectGroup
		id="sessionsPeriod"
		class="w-100"
		:options="periodOptions"
		:modelValue="period"
		@update:modelValue="changePeriod"
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
		period: {
			type: String,
			required: true,
		},
		periodOptions: {
			type: Array,
			required: true,
		},
	},
	methods: {
		changePeriod(newPeriod) {
			this.$emit("update:period", newPeriod);
		},
	},
};
</script>
