<script setup>
import Loadpoints from "./Loadpoints.vue";
import { reactive } from "vue";

function loadpoint(opts) {
	const base = {
		id: 0,
		pvConfigured: true,
		chargePower: 2800,
		chargedEnergy: 11e3,
		chargeDuration: 95 * 60,
		vehiclePresent: true,
		vehicleTitle: "Tesla Model 3",
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
	};
	return { ...base, ...opts };
}

const state = reactive({
	vehicles: [],
	loadpoints: [
		loadpoint({ title: "Carport" }),
		loadpoint({
			title: "Water Heater",
			chargerFeatureIntegratedDevice: true,
			chargerIcon: "waterheater",
		}),
	],
});
</script>

<template>
	<Story>
		<Variant title="standard">
			<Loadpoints v-bind="state" />
		</Variant>
	</Story>
</template>
