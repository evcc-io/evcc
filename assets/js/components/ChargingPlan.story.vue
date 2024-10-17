<script setup>
import ChargingPlan from "./ChargingPlan.vue";

const hoursFromNow = function (hours) {
	const now = new Date();
	now.setHours(now.getHours() + hours);
	return now.toISOString();
};
</script>
<template>
	<Story>
		<Variant title="none">
			<ChargingPlan id="1" />
		</Variant>
		<Variant title="target charge">
			<ChargingPlan id="1" :targetTime="hoursFromNow(14)" />
		</Variant>
		<Variant title="price EUR limit">
			<ChargingPlan
				id="1"
				:smartCostLimit="0.13"
				smartCostType="price"
				currency="EUR"
				:tariffGrid="0.3"
			/>
		</Variant>
		<Variant title="price SEK limit">
			<ChargingPlan
				id="1"
				:smartCostLimit="0.43"
				smartCostType="price"
				currency="SEK"
				:tariffGrid="0.32"
			/>
		</Variant>
		<Variant title="co2 limit">
			<ChargingPlan id="1" :smartCostLimit="400" smartCostType="co2" :tariffCo2="623" />
		</Variant>
		<Variant title="min soc">
			<ChargingPlan id="1" :minSoc="30" :vehicleSoc="25" />
		</Variant>
	</Story>
</template>
