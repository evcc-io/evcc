<script setup>
import { reactive } from "vue";
import SelectGroup from "./SelectGroup.vue";

const state = reactive({
	value: "orange",
});

const options = [
	{ value: "green", name: "Green" },
	{ value: "orange", name: "Orange" },
	{ value: "red", name: "Red" },
];
const controlOptions = options.reduce((res, { value, name }) => {
	res[value] = name;
	return res;
}, {});
</script>

<template>
	<Story auto-props-disabled>
		<Variant title="standard">
			<SelectGroup v-model="state.value" :options="options" />
			<template #controls>
				<HstSelect v-model="state.value" title="value" :options="controlOptions" />
			</template>
		</Variant>
	</Story>
</template>
