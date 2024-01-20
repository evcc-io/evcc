<script setup>
import { reactive } from "vue";
import Vehicle from "./Vehicle.vue";

const state = reactive({
	vehicle: { title: "Mein Auto", icon: "car", capacity: 72, features: [] },
	enabled: false,
	connected: true,
	vehicleName: "meinauto",
	vehicleSoc: 42.742,
	vehicleRange: 231,
	limitSoc: 90,
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
				vehicleName=""
				:socBasedCharging="false"
				:vehicle="{ ...state.vehicle, capacity: null }"
				mode="pv"
			/>
		</Variant>
		<Variant title="offline vehicle">
			<Vehicle
				v-bind="state"
				enabled
				charging
				:socBasedCharging="false"
				:vehicle="{
					...state.vehicle,
					title: 'Opel Corsa-e',
					capacity: 72,
					features: ['Offline'],
				}"
				mode="pv"
			/>
		</Variant>
		<Variant title="offline vehicle with target">
			<Vehicle
				v-bind="state"
				enabled
				charging
				:socBasedCharging="false"
				:vehicle="{
					...state.vehicle,
					title: 'Opel Corsa-e',
					capacity: 72,
					features: ['Offline'],
				}"
				:targetEnergy="30"
				mode="pv"
			/>
		</Variant>
		<Variant title="vehicle limit">
			<Vehicle v-bind="state" enabled charging :vehicleTargetSoc="80" />
		</Variant>
		<Variant title="vehicle limit reached">
			<Vehicle v-bind="state" enabled :vehicleTargetSoc="80" :vehicleSoc="80" />
		</Variant>
		<Variant title="target charge planned">
			<Vehicle v-bind="state" :targetTime="hoursFromNow(34)" mode="pv" />
		</Variant>
		<Variant title="target charge active">
			<Vehicle v-bind="state" enabled charging :targetTime="hoursFromNow(14)" mode="pv" />
		</Variant>
		<Variant title="smart charge cost limit active">
			<Vehicle v-bind="state" enabled charging :smartCostLimit="0.13" mode="pv" />
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
		<Variant title="guard timer">
			<Vehicle v-bind="state" guardAction="enable" :guardRemainingInterpolated="123" />
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
