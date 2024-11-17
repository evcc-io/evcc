<template>
	<div v-for="(plan, index) in plans" :key="index">
		<div>
			<div class="d-lg-none">
				<hr class="w-75 mx-auto mt-5" />
				<h5>
					<div class="inner" data-testid="repeating-plan-title">
						{{ $t("main.chargingPlan.repeatingPlan") + " #" + (index + 2) }}
					</div>
				</h5>
			</div>
			<ChargingPlanRepeatingSettings
				:id="index"
				class="mb-4"
				:formIdPrefix="formIdPrefix"
				v-bind="plan"
				:rangePerSoc="rangePerSoc"
				:numberPlans="numberPlans"
				@repeating-plan-updated="updateRepeatingPlan"
				@repeating-plan-removed="removeRepeatingPlan"
			/>
		</div>
	</div>
	<div class="d-flex align-items-baseline">
		<button
			type="button"
			class="d-flex btn btn-sm btn-outline-secondary border-0 ps-0 align-items-baseline"
			data-testid="repeating-plan-add"
			@click="addRepeatingPlan"
		>
			<shopicon-regular-plus size="s" class="flex-shrink-0 me-2"></shopicon-regular-plus>
			<p class="mb-0">{{ $t("main.chargingPlan.addRepeatingPlan") }}</p>
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
h5 {
	position: relative;
	display: flex;
	top: -25px;
	margin-bottom: -0.5rem;
	padding: 0 0.5rem;
	justify-content: center;
}
h5 .inner {
	padding: 0 0.5rem;
	background-color: var(--evcc-box);
	font-weight: normal;
	color: var(--evcc-gray);
	text-transform: uppercase;
	text-align: center;
}
</style>
