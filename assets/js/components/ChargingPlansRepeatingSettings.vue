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
	<div class="d-flex align-items-center mb-4 pb-2">
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
			return `chargingplan-${this.id}`;
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
			this.plans.push({
				weekdays: DEFAULT_WEEKDAYS,
				time: DEFAULT_TARGET_TIME,
				soc: DEFAULT_TARGET_SOC,
				active: false,
			});
			this.preview();

			// update the plan without storing non-applied changes from other plans
			const plans = [...this.initialPlans]; // clone array
			plans.push({
				weekdays: newPlan.weekdays,
				time: newPlan.time,
				soc: newPlan.soc,
				active: newPlan.active,
			});
			this.updateRepeatingPlans(plans);
			this.updateRepeatingPlans(this.plans);
		},
		updateRepeatingPlans: function (plans) {
			this.$emit("repeating-plans-updated", plans);
		},
		updateRepeatingPlan: function (newPlan) {
			const { id, save } = newPlan;
			this.plans.splice(id, 1, {
				weekdays: newPlan.weekdays,
				time: newPlan.time,
				soc: newPlan.soc,
				active: newPlan.active,
			});
			this.preview();

			if (save) {
				// update the plan without storing non-applied changes from other plans
				const plans = [...this.initialPlans]; // clone array
				plans.splice(id, 1, {
					weekdays: newPlan.weekdays,
					time: newPlan.time,
					soc: newPlan.soc,
					active: newPlan.active,
				});
				this.updateRepeatingPlans(plans);
			}
		},
		removeRepeatingPlan: function (index) {
			this.plans.splice(index, 1);

			// remove the plan without storing non-applied changes from other plans
			const plans = [...this.initialPlans]; // clone array
			plans.splice(index, 1);
			this.updateRepeatingPlans(plans);

			this.preview();
		},
		preview: function () {
			this.$emit("plans-preview", this.plans);
		},
	},
};
</script>

<style scoped>
.btn-outline-secondary {
	margin-left: -0.5rem;
}
</style>
