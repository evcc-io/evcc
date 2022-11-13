<script setup>
import { reactive } from "vue";
import Savings from "./Savings.vue";

const state = reactive({
	totalCharged: 408.88948736396026,
	selfConsumptionPercent: 80.3,
	since: 1666776003,
	amount: 11.924127889686181,
	effectivePrice: 0.22583777338821223,
	gridCharged: 313.4964642464708,
	selfConsumptionCharged: 95.39302311748945,
	gridPrice: 0.34,
	feedInPrice: 0.13,
	currency: "EUR",
});
</script>

<template>
	<Story>
		<Variant title="no sponsor">
			<Savings v-bind="state" />
		</Variant>
		<Variant title="sponsor">
			<Savings v-bind="state" sponsor="naltatis" />
		</Variant>
		<Variant title="no data">
			<Savings />
		</Variant>
	</Story>
</template>
