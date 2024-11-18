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
		<button
			v-if="dataHasChanged"
			type="button"
			class="btn btn-sm btn-outline-primary ms-auto me-4 border-0 text-decoration-underline"
			data-testid="plan-apply"
			@click="updateRepeatingPlans"
		>
			{{ $t("main.chargingPlan.update") }}
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
		dataHasChanged: function () {
			return !deepEqual(this.initialPlans, this.plans);
		},
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
		addRepeatingPlan: function () {
			this.plans.push({
				weekdays: DEFAULT_WEEKDAYS,
				time: DEFAULT_TARGET_TIME,
				soc: DEFAULT_TARGET_SOC,
				active: false,
			});
			this.preview();
		},
		updateRepeatingPlans: function () {
			this.$emit("repeating-plans-updated", this.plans);
		},
		updateRepeatingPlan: function (newPlan) {
			const { id } = newPlan;
			this.plans.splice(id, 1, {
				weekdays: newPlan.weekdays,
				time: newPlan.time,
				soc: newPlan.soc,
				active: newPlan.active,
			});
			this.preview();
		},
		removeRepeatingPlan: function (index) {
			this.plans.splice(index, 1);
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
