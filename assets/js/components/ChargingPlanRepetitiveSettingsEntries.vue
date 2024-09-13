<template>
	<div v-for="(plan, index) in plans" :key="index">
		<ChargingPlanRepetitiveSettingsEntry
			class="mb-4"
			:id="index"
			v-bind="plan"
			:socBasedPlanning="socBasedPlanning"
			:rangePerSoc="rangePerSoc"
			@repetitive-plan-removed="removeRepetitivePlan"
		/>
	</div>
	<button
		type="button"
		class="d-flex btn btn-sm btn-outline-primary border-0 ps-0"
		data-testid="repetitive-plan-add"
		@click="addRepetitivePlan"
	>
		<shopicon-regular-plus size="s" class="flex-shrink-0 me-2"></shopicon-regular-plus>
		<p class="mb-0">{{ $t("main.chargingPlan.addRepetitivePlan") }}</p>
	</button>
</template>

<script>
import api from "../api";
import ChargingPlanRepetitiveSettingsEntry from "./ChargingPlanRepetitiveSettingsEntry.vue";

const DEFAULT_WEEKDAYS = [0];
const DEFAULT_TARGET_TIME = "12:00";
const DEFAULT_TARGET_SOC = 80;

export default {
	name: "ChargingPlanRepetitiveSettingsEntries",
	components: {
		ChargingPlanRepetitiveSettingsEntry,
	},
	props: {
		id: Number,
		socBasedPlanning: Boolean,
		rangePerSoc: Number,
	},
	data: function () {
		return {
			plans: [],
		};
	},
	mounted() {
		this.fetchRepetitivePlans();
	},
	methods: {
		fetchRepetitivePlans: async function () {
			let response = await api.get(`/loadpoints/${this.id}/plan/repetitive`);
			this.plans = response.data.result;
		},
		addRepetitivePlan: function () {
			this.plans.push({
				weekdays: DEFAULT_WEEKDAYS,
				time: DEFAULT_TARGET_TIME,
				soc: DEFAULT_TARGET_SOC,
				active: false,
			});
		},
		removeRepetitivePlan: function (index) {
			this.plans.splice(index, 1);
		},
	},
};
</script>
