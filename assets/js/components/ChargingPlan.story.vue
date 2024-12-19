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
		<Variant title="min soc">
			<ChargingPlan id="1" :minSoc="30" :vehicleSoc="25" />
		</Variant>
	</Story>
</template>
