<script setup>
import { reactive } from "vue";
import Vehicle from "./Vehicle.vue";

const state = reactive({
	vehicleTitle: "Mein Auto",
	enabled: false,
	connected: true,
	vehiclePresent: true,
	vehicleSoC: 42,
	vehicleRange: 231,
	targetSoC: 90,
	vehicleCapacity: 72,
	chargedEnergy: 14123,
	socBasedCharging: true,
	id: 0,
});

const hoursFromNow = function (hours) {
	const now = new Date();
	now.setHours(now.getHours() + hours);
	return now.toISOString();
};
</script>

<template>
	<Story>
		<Variant title="disconnected">
			<Vehicle v-bind="state" :connected="false" />
		</Variant>
		<Variant title="connected">
			<Vehicle v-bind="state" />
		</Variant>
		<Variant title="ready">
			<Vehicle v-bind="state" enabled />
		</Variant>
		<Variant title="charging">
			<Vehicle v-bind="state" enabled charging />
		</Variant>
		<Variant title="charging">
			<Vehicle v-bind="state" enabled charging />
		</Variant>
		<Variant title="unknown vehicle">
			<Vehicle
				v-bind="state"
				enabled
				charging
				:vehiclePresent="false"
				:socBasedCharging="false"
				:vehicleCapacity="null"
				mode="pv"
			/>
		</Variant>
		<Variant title="offline vehicle">
			<Vehicle
				v-bind="state"
				enabled
				charging
				vehicleTitle="Opel Corsa-e"
				:socBasedCharging="false"
				:vehicleCapacity="72"
				vehicleFeatureOffline
				mode="pv"
			/>
		</Variant>
		<Variant title="offline vehicle with target">
			<Vehicle
				v-bind="state"
				enabled
				charging
				vehicleTitle="Opel Corsa-e"
				:socBasedCharging="false"
				:vehicleCapacity="72"
				:targetEnergy="30"
				vehicleFeatureOffline
				mode="pv"
			/>
		</Variant>
		<Variant title="vehicle limit">
			<Vehicle v-bind="state" enabled charging :vehicleTargetSoC="80" />
		</Variant>
		<Variant title="vehicle limit reached">
			<Vehicle v-bind="state" enabled :vehicleTargetSoC="80" :vehicleSoC="80" />
		</Variant>
		<Variant title="target charge planned">
			<Vehicle v-bind="state" :targetTime="hoursFromNow(14)" mode="pv" />
		</Variant>
		<Variant title="target charge active">
			<Vehicle v-bind="state" enabled charging :targetTime="hoursFromNow(14)" mode="pv" />
		</Variant>
		<Variant title="pv enable timer">
			<Vehicle v-bind="state" pvAction="enable" :pvRemainingInterpolated="32" />
		</Variant>
		<Variant title="pv disable timer">
			<Vehicle
				v-bind="state"
				enabled
				charging
				pvAction="disable"
				:pvRemainingInterpolated="155"
			/>
		</Variant>
		<Variant title="vehicle switch">
			<Vehicle
				v-bind="state"
				vehicleTitle="Blauer e-Golf"
				:vehicles="['Blauer e-Golf', 'Weißes Model 3']"
			/>
		</Variant>
	</Story>
</template>
