<template>
	<div v-for="(plan, index) in plans" :key="index">
		<ChargingPlanRepeatingSettings
			class="mb-4"
			:id="index"
			v-bind="plan"
			:socBasedPlanning="socBasedPlanning"
			:rangePerSoc="rangePerSoc"
			@repeating-plan-updated="updateRepeatingPlan"
			@repeating-plan-removed="removeRepeatingPlan"
		/>
	</div>
	<div class="d-flex align-items-baseline">
		<button
			type="button"
			class="d-flex btn btn-sm btn-outline-primary border-0 ps-0 align-items-baseline"
			data-testid="repeating-plan-add"
			@click="addRepeatingPlan"
		>
			<shopicon-regular-plus size="s" class="flex-shrink-0 me-2"></shopicon-regular-plus>
			<p class="mb-0">{{ $t("main.chargingPlan.addRepeatingPlan") }}</p>
		</button>
		<button
			v-if="dataHasChanged"
			type="button"
			class="btn btn-sm btn-outline-primary ms-3 border-0 text-decoration-underline"
			data-testid="plan-apply"
			@click="updateRepeatingPlans"
		>
			{{ $t("main.chargingPlan.update") }}
		</button>
	</div>
</template>

<script>
import ChargingPlanRepeatingSettings from "./ChargingPlanRepeatingSettings.vue";

const DEFAULT_WEEKDAYS = [0];
const DEFAULT_TARGET_TIME = "12:00";
const DEFAULT_TARGET_SOC = 80;

export default {
	name: "ChargingPlansRepeatingSettings",
	components: {
		ChargingPlanRepeatingSettings,
	},
	props: {
		id: Number,
		socBasedPlanning: Boolean,
		rangePerSoc: Number,
	},
	emits: ["repeating-plans-updated"],
	data: function () {
		return {
			plans: [],
			initialPlans: [],
		};
	},
	computed: {
		dataHasChanged: function () {
			return JSON.stringify(this.initialPlans) !== JSON.stringify(this.plans);
		},
	},
	methods: {
		addRepeatingPlan: function () {
			this.plans.push({
				weekdays: DEFAULT_WEEKDAYS,
				time: DEFAULT_TARGET_TIME,
				soc: DEFAULT_TARGET_SOC,
				active: false,
			});
		},
		updateRepeatingPlans: function () {
			this.initialPlans = [...this.plans]; // clone array
			this.$emit("repeating-plans-updated", this.initialPlans);
		},
		updateRepeatingPlan: function (newData) {
			this.plans[newData.id] = {
				weekdays: newData.weekdays,
				time: newData.time,
				soc: newData.soc,
				active: newData.active,
			};
		},
		removeRepeatingPlan: function (index) {
			this.plans.splice(index, 1);
		},
	},
};
</script>
