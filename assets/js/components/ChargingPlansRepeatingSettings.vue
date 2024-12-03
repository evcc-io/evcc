<template>
	<div v-for="(plan, index) in plans" :key="index" data-testid="plan-entry">
		<div>
			<ChargingPlanRepeatingSettings
				:showHeader="index === 0"
				:number="index + 2"
				class="mb-5 mb-lg-4"
				:formIdPrefix="formIdPrefix"
				v-bind="plan"
				:rangePerSoc="rangePerSoc"
				@updated="updatePlan(index, $event)"
				@saved="savePlan(index, $event)"
				@removed="removePlan(index)"
			/>
		</div>
	</div>
	<div class="d-flex align-items-center pb-4">
		<button
			type="button"
			class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray"
			data-testid="repeating-plan-add"
			tabindex="0"
			@click="addPlan"
		>
			<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
			{{ $t("main.chargingPlan.addRepeatingPlan") }}
		</button>
	</div>
</template>

<script>
import ChargingPlanRepeatingSettings from "./ChargingPlanRepeatingSettings.vue";
import deepEqual from "../utils/deepEqual";
import formatter from "../mixins/formatter";

const DEFAULT_WEEKDAYS = [1, 2, 3, 4, 5]; // weekdays
const DEFAULT_TARGET_TIME = "07:00";
const DEFAULT_TARGET_SOC = 80;

export default {
	name: "ChargingPlansRepeatingSettings",
	components: {
		ChargingPlanRepeatingSettings,
	},
	mixins: [formatter],
	props: {
		id: Number,
		rangePerSoc: Number,
		plans: { type: Array, default: () => [] },
	},
	emits: ["updated"],
	computed: {
		formIdPrefix: function () {
			return `chargingplan-lp${this.id}`;
		},
	},
	methods: {
		deepEqual,
		addPlan: function () {
			const newPlan = {
				weekdays: DEFAULT_WEEKDAYS,
				time: DEFAULT_TARGET_TIME,
				soc: DEFAULT_TARGET_SOC,
				active: false,
				tz: this.timezone(),
			};

			// update the plan without storing non-applied changes from other plans
			const plans = [...this.plans]; // clone array
			plans.push(newPlan);
			this.updatePlans(plans);
		},
		updatePlan: function (index, plan) {
			const plans = [...this.plans]; // clone array
			plans.splice(index, 1, plan);
			this.updatePlans(plans);
		},
		updatePlans: function (plans) {
			this.$emit("updated", plans);
		},
		removePlan: function (index) {
			const plans = [...this.plans]; // clone array
			plans.splice(index, 1);
			this.updatePlans(plans);
		},
	},
};
</script>

<style scoped>
.btn-outline-secondary {
	margin-left: -0.5rem;
}
</style>
