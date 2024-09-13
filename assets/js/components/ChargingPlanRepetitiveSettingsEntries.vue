<template>
	<div v-for="(plan, index) in plans" :key="index">
		<ChargingPlanRepetitiveSettingsEntry
			class="mb-4"
			:id="index"
			v-bind="plan"
			:socBasedPlanning="socBasedPlanning"
			:rangePerSoc="rangePerSoc"
			@repetitive-plan-updated="updateRepetitivePlan"
			@repetitive-plan-removed="removeRepetitivePlan"
		/>
	</div>
	<div class="d-flex align-items-baseline">
		<button
			type="button"
			class="d-flex btn btn-sm btn-outline-primary border-0 ps-0 align-items-baseline"
			data-testid="repetitive-plan-add"
			@click="addRepetitivePlan"
		>
			<shopicon-regular-plus size="s" class="flex-shrink-0 me-2"></shopicon-regular-plus>
			<p class="mb-0">{{ $t("main.chargingPlan.addRepetitivePlan") }}</p>
		</button>
		<button
			v-if="hasDataChanged"
			type="button"
			class="btn btn-sm btn-outline-primary ms-3 border-0 text-decoration-underline"
			data-testid="plan-apply"
			@click="update"
		>
			{{ $t("main.chargingPlan.update") }}
		</button>
	</div>
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
			initialPlans: [],
		};
	},
	mounted() {
		this.fetchRepetitivePlans();
	},
	computed: {
		hasDataChanged: function () {
			return JSON.stringify(this.initialPlans) !== JSON.stringify(this.plans);
		},
	},
	methods: {
		fetchRepetitivePlans: async function () {
			let response = await api.get(`/loadpoints/${this.id}/plan/repetitive`);
			this.plans = response.data.result;
			this.initialPlans = [...response.data.result]; // clone array
		},
		addRepetitivePlan: function () {
			this.plans.push({
				weekdays: DEFAULT_WEEKDAYS,
				time: DEFAULT_TARGET_TIME,
				soc: DEFAULT_TARGET_SOC,
				active: false,
			});
		},
		update: async function () {
			// TODO: update data
			this.initialPlans = [...this.plans];
		},
		updateRepetitivePlan: function (newData) {
			this.plans[newData.id] = {
				weekdays: newData.weekdays,
				time: newData.time,
				soc: newData.soc,
				active: newData.active,
			};
		},
		removeRepetitivePlan: function (index) {
			this.plans.splice(index, 1);
		},
	},
};
</script>
