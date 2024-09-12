<template>
	<div v-for="(plan, index) in plans" :key="index">
		<ChargingPlanRepetitiveSettingsEntry
			class="mb-4"
			:id="index"
			:weekdays="plan.weekdays"
			:time="plan.time"
			:soc="plan.soc"
			:active="plan.active"
			:socBasedPlanning="socBasedPlanning"
		/>
	</div>
</template>

<script>
import ChargingPlanRepetitiveSettingsEntry from "./ChargingPlanRepetitiveSettingsEntry.vue";

export default {
	name: "ChargingPlanRepetitiveSettingsEntries",
	components: {
		ChargingPlanRepetitiveSettingsEntry,
	},
	props: {
		id: String,
		socBasedPlanning: Boolean,
	},
	data: function () {
		return {
			plans: [
				{ weekdays: [0, 3, 4], time: "10:15", soc: 80, active: true },
				{ weekdays: [4], time: "10:15", soc: 40, active: false },
			],
		};
	},
};
</script>
