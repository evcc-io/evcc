<script setup>
import { reactive } from "vue";
import Mode from "./Mode.vue";

const state = reactive({
	mode: "pv",
});
</script>

<template>
	<Story auto-props-disabled>
		<Variant title="standard">
			<Mode :mode="state.mode" />
			<template #controls>
				<HstSelect
					v-model="state.mode"
					title="mode"
					:options="{
						off: 'off',
						now: 'now',
						minpv: 'minpv',
						pv: 'pv',
					}"
				/>
			</template>
		</Variant>
	</Story>
</template>
