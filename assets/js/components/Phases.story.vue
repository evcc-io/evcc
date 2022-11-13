<script setup>
import Phases from "./Phases.vue";
import { reactive } from "vue";

const state = reactive({
	phasesActive: 1,
	minCurrent: 6,
	maxCurrent: 16,
	chargeCurrent: 8,
	chargeCurrents: null,
});
</script>

<template>
	<Story :layout="{ type: 'grid', width: 300 }">
		<Variant title="one phase">
			<Phases v-bind="state" />
		</Variant>
		<Variant title="two phase">
			<Phases v-bind="state" :phasesActive="2" />
		</Variant>
		<Variant title="three phase">
			<Phases v-bind="state" :phasesActive="3" />
		</Variant>
		<Variant title="real currents">
			<Phases
				v-bind="state"
				:phasesActive="3"
				:chargeCurrent="13"
				:chargeCurrents="[11, 9, 12]"
			/>
		</Variant>
		<Variant title="more available">
			<Phases v-bind="state" :chargeCurrent="12" :chargeCurrents="[6, 0, 0]" />
		</Variant>
	</Story>
</template>
