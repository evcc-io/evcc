<script setup>
import SmartCharging from "./SmartCharging.vue";

const hoursFromNow = function (hours) {
	const now = new Date();
	now.setHours(now.getHours() + hours);
	return now.toISOString();
};
</script>
<template>
	<Story>
		<Variant title="none">
			<SmartCharging id="1" />
		</Variant>
		<Variant title="target charge">
			<SmartCharging id="1" :targetTime="hoursFromNow(14)" />
		</Variant>
		<Variant title="price EUR limit">
			<SmartCharging id="1" :smartCostLimit="0.13" smartCostUnit="EUR" :tariffGrid="0.3" />
		</Variant>
		<Variant title="price SEK limit">
			<SmartCharging id="1" :smartCostLimit="0.43" smartCostUnit="SEK" :tariffGrid="0.32" />
		</Variant>
		<Variant title="co2 limit">
			<SmartCharging id="1" :smartCostLimit="400" smartCostUnit="gCO2eq" :tariffCo2="623" />
		</Variant>
		<Variant title="min soc">
			<SmartCharging id="1" :minSoc="30" :vehicleSoc="25" />
		</Variant>
	</Story>
</template>
