<script setup lang="ts">
import { reactive } from "vue";
import SelectGroup from "./SelectGroup.vue";
import type { SelectOption } from "@/types/evcc";

const state = reactive({
	value: "orange",
});

const options: SelectOption<string>[] = [
	{ value: "green", name: "Green" },
	{ value: "orange", name: "Orange" },
	{ value: "red", name: "Red" },
];
const controlOptions = options.reduce((res: Record<string, string>, { value, name }) => {
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
