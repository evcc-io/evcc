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
	vehicleSoC: 66,
	targetSoC: 90,
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
		<Variant title="disabled">
			<Loadpoint
				v-bind="state"
				remoteDisabled="soft"
				remoteDisabledSource="Sunny Home Manager"
				mode="now"
				:enabled="false"
				:charging="false"
				:chargePower="0"
			/>
		</Variant>
	</Story>
</template>
