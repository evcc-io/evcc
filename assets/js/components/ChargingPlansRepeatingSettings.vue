<template>
	<div v-for="(plan, index) in plans" :key="index" :data-testid="`repeating-plan-${index + 2}`">
		<div>
			<ChargingPlanRepeatingSettings
				:id="index"
				class="mb-5 mb-lg-4"
				:formIdPrefix="formIdPrefix"
				v-bind="plan"
				:rangePerSoc="rangePerSoc"
				:numberPlans="numberPlans"
				:dataChanged="!deepEqual(initialPlans[index], plan)"
				@repeating-plan-updated="updateRepeatingPlan"
				@repeating-plan-removed="removeRepeatingPlan"
			/>
		</div>
	</div>
	<div class="d-flex align-items-center pb-4">
		<button
			type="button"
			class="d-flex btn btn-sm btn-outline-secondary border-0 align-items-center gap-2 evcc-gray"
			data-testid="repeating-plan-add"
			@click="addRepeatingPlan"
		>
			<shopicon-regular-plus size="s" class="flex-shrink-0"></shopicon-regular-plus>
			{{ $t("main.chargingPlan.addRepeatingPlan") }}
		</button>
	</div>
</template>

<script>
import ChargingPlanRepeatingSettings from "./ChargingPlanRepeatingSettings.vue";
import deepEqual from "../utils/deepEqual";

const DEFAULT_WEEKDAYS = [1]; // Monday
const DEFAULT_TARGET_TIME = "07:00";
const DEFAULT_TARGET_SOC = 80;

export default {
	name: "ChargingPlansRepeatingSettings",
	components: {
		ChargingPlanRepeatingSettings,
	},
	props: {
		id: Number,
		rangePerSoc: Number,
		initialPlans: { type: Array, default: () => [] },
		numberPlans: Boolean,
	},
	emits: ["repeating-plans-updated", "plans-preview"],
	data: function () {
		return {
			plans: [...this.initialPlans], // clone array
		};
	},
	computed: {
		formIdPrefix: function () {
			return `chargingplan-lp${this.id}`;
		},
	},
	watch: {
		initialPlans(newPlans) {
			if (deepEqual(newPlans, this.plans)) {
				this.plans = [...newPlans]; // clone array
			}
		},
	},
	methods: {
		deepEqual,
		addRepeatingPlan: function () {
			const newPlan = {
				weekdays: DEFAULT_WEEKDAYS,
				time: DEFAULT_TARGET_TIME,
				soc: DEFAULT_TARGET_SOC,
				active: false,
			};
			this.plans.push(newPlan);

			// update the plan without storing non-applied changes from other plans
			const plans = [...this.initialPlans]; // clone array
			plans.push(newPlan);
			this.updateRepeatingPlans(plans);
			this.preview(this.id);
		},
		updateRepeatingPlans: function (plans) {
			this.$emit("repeating-plans-updated", plans);
		},
		updateRepeatingPlan: function (newPlanData) {
			const { id, save, preview, plan } = newPlanData;
			this.plans.splice(id, 1, plan);

			if (save) {
				// update the plan without storing non-applied changes from other plans
				const plans = [...this.initialPlans]; // clone array
				plans.splice(id, 1, plan);
				this.updateRepeatingPlans(plans);
			}

			if (preview) {
				this.preview(id);
			}
		},
		removeRepeatingPlan: function (index) {
			this.plans.splice(index, 1);

			// remove the plan without storing non-applied changes from other plans
			const plans = [...this.initialPlans]; // clone array
			plans.splice(index, 1);
			this.updateRepeatingPlans(plans);

			this.preview(index);
		},
		preview: function (index) {
			this.$emit("plans-preview", {plans: this.plans, index: index});
		},
	},
};
</script>

<style scoped>
.btn-outline-secondary {
	margin-left: -0.5rem;
}
</style>
