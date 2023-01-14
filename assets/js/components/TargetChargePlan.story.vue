<script setup>
import TargetChargePlan from "./TargetChargePlan.vue";

const priceSlots1 = [
	{ price: 0.41, start: "2022-12-22T19:00:00Z" },
	{ price: 0.22, start: "2022-12-22T20:00:00Z" },
	{ price: 0.19, start: "2022-12-22T21:00:00Z" },
	{ price: 0.07, start: "2022-12-22T22:00:00Z" },
	{ price: 0.33, start: "2022-12-22T23:00:00Z" },
	{ price: 0.34, start: "2022-12-23T00:00:00Z" },
	{ price: 0.61, start: "2022-12-23T01:00:00Z" },
	{ price: 0.57, start: "2022-12-23T02:00:00Z" },
	{ price: 0.42, start: "2022-12-23T03:00:00Z" },
];

const chargingSlots = [{ start: "2022-12-22T21:00:00Z", end: "2022-12-22T23:00:00Z" }];
</script>

<template>
	<Story title="TargetChargePlan">
		<Variant title="standard">
			<TargetChargePlan
				:price-slots="priceSlots1"
				:charging-slots="chargingSlots"
				currency="EUR"
			/>
		</Variant>
	</Story>
</template>
