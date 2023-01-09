<script setup>
import BatteryIcon from "./BatteryIcon.vue";
</script>

<template>
	<Story :layout="{ type: 'grid', width: 150 }">
		<Variant title="empty"> <BatteryIcon :soc="10" /> </Variant>
		<Variant title="quarter"> <BatteryIcon :soc="30" /> </Variant>
		<Variant title="half"> <BatteryIcon :soc="50" /> </Variant>
		<Variant title="three quarter"> <BatteryIcon :soc="70" /> </Variant>
		<Variant title="full"> <BatteryIcon :soc="90" /> </Variant>
	</Story>
</template>
