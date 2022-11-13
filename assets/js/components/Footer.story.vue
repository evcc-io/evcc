<script setup>
import { reactive } from "vue";
import Footer from "./Footer.vue";

const timeAgo = (hours = 0, minutes = 0, seconds = 0) => {
	const date = new Date();
	date.setHours(date.getHours() - hours);
	date.setMinutes(date.getMinutes() - minutes);
	date.setSeconds(date.getSeconds() - seconds);
	return date;
};

const state = reactive({
	version: {
		installed: "0.401",
		available: "0.401",
		commit: "5ce7be4",
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
	</Story>
</template>
