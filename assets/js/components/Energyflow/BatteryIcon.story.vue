<script setup>
import BatteryIcon from "./BatteryIcon.vue";
</script>

<template>
	<Story :layout="{ type: 'grid', width: 150 }">
		<Variant title="0%"> <BatteryIcon :soc="0" /> </Variant>
		<Variant title="10%"> <BatteryIcon :soc="10" /> </Variant>
		<Variant title="20%"> <BatteryIcon :soc="20" /> </Variant>
		<Variant title="30%"> <BatteryIcon :soc="30" /> </Variant>
		<Variant title="40%"> <BatteryIcon :soc="40" /> </Variant>
		<Variant title="50%"> <BatteryIcon :soc="50" /> </Variant>
		<Variant title="60%"> <BatteryIcon :soc="60" /> </Variant>
		<Variant title="70%"> <BatteryIcon :soc="70" /> </Variant>
		<Variant title="80%"> <BatteryIcon :soc="80" /> </Variant>
		<Variant title="90%"> <BatteryIcon :soc="90" /> </Variant>
		<Variant title="hold"> <BatteryIcon hold /> </Variant>
		<Variant title="grid charge"> <BatteryIcon grid-charge /> </Variant>
	</Story>
</template>
