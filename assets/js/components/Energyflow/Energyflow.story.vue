<script setup>
import Energyflow from "./Energyflow.vue";
</script>

<template>
	<Story>
		<Variant title="grid and pv">
			<Energyflow
				gridConfigured
				pvConfigured
				:pvPower="7300"
				:gridPower="-2300"
				:homePower="800"
				:loadpointsPower="4200"
				:activeLoadpointsCount="3"
				siteTitle="Home"
			/>
		</Variant>
		<Variant title="battery and grid">
			<Energyflow
				gridConfigured
				pvConfigured
				batteryConfigured
				:pvPower="0"
				:gridPower="1200"
				:homePower="2000"
				:batteryPower="800"
				:batterySoC="77"
				siteTitle="Home"
			/>
		</Variant>
		<Variant title="battery charging">
			<Energyflow
				gridConfigured
				pvConfigured
				batteryConfigured
				:pvPower="5000"
				:gridPower="-1300"
				:homePower="800"
				:loadpointsPower="1400"
				:activeLoadpointsCount="1"
				:batteryPower="-1500"
				:batterySoC="75"
				siteTitle="Home"
			/>
		</Variant>
		<Variant title="grid, pv and battery">
			<Energyflow
				gridConfigured
				pvConfigured
				batteryConfigured
				:pvPower="1000"
				:gridPower="700"
				:homePower="3300"
				:batteryPower="1500"
				:batterySoC="30"
				siteTitle="Home"
			/>
		</Variant>
		<Variant title="battery thresholds">
			<Energyflow
				gridConfigured
				pvConfigured
				batteryConfigured
				:pvPower="8700"
				:gridPower="-500"
				:loadpointsPower="7500"
				:activeLoadpointsCount="2"
				:batteryPower="-700"
				:batterySoC="95"
				siteTitle="Home"
			/>
		</Variant>
		<Variant title="pv thresholds">
			<Energyflow
				gridConfigured
				pvConfigured
				batteryConfigured
				:pvPower="300"
				:gridPower="5500"
				:homePower="1000"
				:loadpointsPower="5600"
				:activeLoadpointsCount="2"
				:batteryPower="800"
				:batterySoC="76"
				siteTitle="Home"
			/>
		</Variant>
		<Variant title="grid only">
			<Energyflow
				gridConfigured
				pvConfigured
				batteryConfigured
				:pvPower="0"
				:gridPower="6500"
				:homePower="1000"
				:loadpointsPower="5500"
				:activeLoadpointsCount="1"
				:batteryPower="0"
				:batterySoC="0"
				siteTitle="Home"
			/>
		</Variant>
		<Variant title="low power">
			<Energyflow
				gridConfigured
				pvConfigured
				batteryConfigured
				:pvPower="700"
				:gridPower="-300"
				:homePower="300"
				:batteryPower="-100"
				:batterySoC="55"
				siteTitle="Home"
			/>
		</Variant>
	</Story>
</template>
