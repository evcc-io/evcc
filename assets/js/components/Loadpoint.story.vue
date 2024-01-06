<script setup>
import Loadpoint from "./Loadpoint.vue";
import { reactive } from "vue";

const state = reactive({
	id: 0,
	pvConfigured: true,
	chargePower: 2800,
	chargedEnergy: 11e3,
	chargeDuration: 95 * 60,
	vehiclePresent: true,
	vehicleTitle: "Mein Auto",
	enabled: true,
	connected: true,
	mode: "pv",
	charging: true,
	vehicleSoc: 66,
	limitSoc: 90,
	chargeCurrent: 7,
	minCurrent: 6,
	maxCurrent: 16,
	activePhases: 2,
});
</script>

<template>
	<Story>
		<Variant title="standard">
			<Loadpoint v-bind="state" />
		</Variant>
		<Variant title="without soc">
			<Loadpoint v-bind="state" vehicleTitle="" :vehiclePresent="false" />
		</Variant>
		<Variant title="idle">
			<Loadpoint
				v-bind="state"
				:enabled="false"
				:connected="false"
				:vehiclePresent="false"
				mode="off"
				:charging="false"
				:chargeCurrent="0"
			/>
		</Variant>
		<Variant title="disabled, long title">
			<Loadpoint
				v-bind="state"
				title="Charging point with a very very very long title!!!1!"
				remoteDisabled="soft"
				remoteDisabledSource="Sunny Home Manager"
				mode="now"
				:enabled="false"
				:charging="false"
				:chargePower="0"
			/>
		</Variant>
		<Variant title="charger icon, no vehicle">
			<Loadpoint
				v-bind="state"
				chargerIcon="heater"
				title="Heating device with long name"
				mode="now"
				chargerFeatureIntegratedDevice
			/>
		</Variant>
	</Story>
</template>
