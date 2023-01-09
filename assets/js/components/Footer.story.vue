<script setup>
import { reactive } from "vue";
import Footer from "./Footer.vue";

const state = reactive({
	version: {
		installed: "0.401",
		available: "0.401",
	},
	savings: {
		selfConsumptionPercent: 88,
	},
});
</script>

<template>
	<Story>
		<Variant title="default">
			<Footer v-bind="state" />
		</Variant>
		<Variant title="update nightly">
			<Footer
				v-bind="state"
				:version="{ installed: '0.400', available: '0.400', commit: '5ce7be4' }"
			/>
		</Variant>
		<Variant title="update availabe">
			<Footer v-bind="state" :version="{ installed: '0.400', available: '0.500' }" />
		</Variant>
	</Story>
</template>
