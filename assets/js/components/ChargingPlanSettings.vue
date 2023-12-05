<template>
	<div class="mt-4">
		<div class="form-group d-lg-flex align-items-baseline justify-content-between">
			<div v-if="plans.length > 0" class="container px-0 mb-3">
				<ChargingPlanSettingsEntry
					v-for="(p, index) in plans"
					:id="`${id}_${index}`"
					:key="index"
					class="mb-2"
					v-bind="p"
					:vehicle-capacity="vehicleCapacity"
					:range-per-soc="rangePerSoc"
					:soc-per-kwh="socPerKwh"
					:soc-based-planning="socBasedPlanning"
					@plan-updated="(data) => updatePlan({ index, ...data })"
					@plan-removed="() => removePlan(index)"
				/>
			</div>
			<div v-else>
				<p>
					{{ $t("main.targetCharge.planDescription") }}
				</p>
				<button class="btn btn-outline-primary" type="button" @click="addPlan">
					{{ $t("main.targetCharge.setPlan") }}
				</button>
			</div>
		</div>
		<div v-if="plans.length > 0">
			<ChargingPlanWarnings v-bind="chargingPlanWarningsProps" class="mb-4" />
			<hr />
			<h5>PREVIEW</h5>
			<ChargingPlanPreview
				v-if="chargingPlanPreviewProps"
				v-bind="chargingPlanPreviewProps"
			/>
		</div>
	</div>
</template>

<script>
import ChargingPlanPreview from "./ChargingPlanPreview.vue";
import ChargingPlanSettingsEntry from "./ChargingPlanSettingsEntry.vue";
import ChargingPlanWarnings from "./ChargingPlanWarnings.vue";
import formatter from "../mixins/formatter";
import collector from "../mixins/collector";
import api from "../api";

const DEFAULT_TARGET_TIME = "7:00";
const LAST_TARGET_TIME_KEY = "last_target_time";

export default {
	name: "ChargingPlanSettings",
	components: { ChargingPlanPreview, ChargingPlanSettingsEntry, ChargingPlanWarnings },
	mixins: [formatter, collector],
	props: {
		id: [String, Number],
		plans: { type: Array, default: () => [] },
		effectiveLimitSoc: Number,
		effectivePlanTime: String,
		effectivePlanSoc: Number,
		planEnergy: Number,
		limitEnergy: Number,
		socBasedPlanning: Boolean,
		socPerKwh: Number,
		rangePerSoc: Number,
		smartCostLimit: Number,
		smartCostType: String,
		currency: String,
		mode: String,
		vehicleCapacity: Number,
		vehicle: Object,
	},
	emits: ["plan-removed", "plan-updated"],
	data: function () {
		return {
			tariff: {},
			plan: {},
			activeTab: "time",
			loading: false,
		};
	},
	computed: {
		chargingPlanWarningsProps: function () {
			return this.collectProps(ChargingPlanWarnings);
		},
		selectedTargetTime: function () {
			return new Date(this.effectivePlanTime);
		},
		chargingPlanPreviewProps: function () {
			const targetTime = this.selectedTargetTime;
			const { rates } = this.tariff;
			const { duration, plan, power } = this.plan;
			const { currency, smartCostType } = this;
			return rates
				? { duration, plan, power, rates, targetTime, currency, smartCostType }
				: null;
		},
	},
	watch: {
		plans: {
			handler() {
				this.fetchPlan();
			},
			deep: true,
		},
	},
	mounted() {
		this.fetchPlan();
	},
	methods: {
		fetchPlan: async function () {
			if (this.plans.length > 0 && !this.loading) {
				try {
					this.loading = true;
					this.plan = (await api.get(`loadpoints/${this.id}/plan`)).data.result;

					const tariffRes = await api.get(`tariff/planner`, {
						validateStatus: function (status) {
							return status >= 200 && status < 500;
						},
					});
					this.tariff = tariffRes.status === 404 ? { rates: [] } : tariffRes.data.result;
				} catch (e) {
					console.error(e);
				} finally {
					this.loading = false;
				}
			}
		},
		defaultDate: function () {
			const [hours, minutes] = (
				window.localStorage[LAST_TARGET_TIME_KEY] || DEFAULT_TARGET_TIME
			).split(":");

			const target = new Date();
			target.setSeconds(0);
			target.setMinutes(minutes);
			target.setHours(hours);
			// today or tomorrow?
			const isInPast = target < new Date();
			if (isInPast) {
				target.setDate(target.getDate() + 1);
			}
			return target;
		},
		addPlan: function () {
			this.$emit("plan-updated", {
				time: this.defaultDate(),
				soc: 100,
				energy: this.vehicleCapacity || 10,
			});
		},
		removePlan: function (index) {
			this.$emit("plan-removed", index);
		},
		updatePlan: function (data) {
			this.$emit("plan-updated", data);
		},
	},
};
</script>

<style scoped>
h5 {
	position: relative;
	display: inline-block;
	background-color: var(--evcc-box);
	top: -25px;
	left: calc(50% - 50px);
	padding: 0 0.5rem;
	font-weight: normal;
	color: var(--evcc-gray);
	margin-bottom: -4rem;
}
</style>
