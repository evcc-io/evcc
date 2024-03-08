<template>
	<div class="mt-4">
		<div class="form-group d-lg-flex align-items-baseline justify-content-between">
			<div class="container px-0 mb-3">
				<ChargingPlanSettingsEntry
					:id="`${id}_0`"
					class="mb-2"
					v-bind="plans[0] || {}"
					:capacity="capacity"
					:range-per-soc="rangePerSoc"
					:soc-per-kwh="socPerKwh"
					:soc-based-planning="socBasedPlanning"
					@plan-updated="(data) => updatePlan({ index: 0, ...data })"
					@plan-removed="() => removePlan(0)"
					@plan-preview="previewPlan"
				/>
			</div>
		</div>
		<ChargingPlanWarnings v-bind="chargingPlanWarningsProps" class="mb-4" />
		<hr />
		<h5>
			<div class="inner" data-testid="plan-preview-title">
				{{ $t(`main.targetCharge.${isPreview ? "preview" : "currentPlan"}`) }}
			</div>
		</h5>
		<ChargingPlanPreview v-bind="chargingPlanPreviewProps" />
	</div>
</template>

<script>
import ChargingPlanPreview from "./ChargingPlanPreview.vue";
import ChargingPlanSettingsEntry from "./ChargingPlanSettingsEntry.vue";
import ChargingPlanWarnings from "./ChargingPlanWarnings.vue";
import formatter from "../mixins/formatter";
import collector from "../mixins/collector";
import api from "../api";
import deepEqual from "../utils/deepEqual";

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
		capacity: Number,
		vehicle: Object,
		vehicleTargetSoc: Number,
	},
	emits: ["plan-removed", "plan-updated"],
	data: function () {
		return {
			tariff: {},
			plan: {},
			activeTab: "time",
			isPreview: false,
			debounceTimer: null,
		};
	},
	computed: {
		chargingPlanWarningsProps: function () {
			return this.collectProps(ChargingPlanWarnings);
		},
		chargingPlanPreviewProps: function () {
			const { rates } = this.tariff;
			const { duration, plan, power, planTime } = this.plan;
			const targetTime = planTime ? new Date(planTime) : null;
			const { currency, smartCostType } = this;
			return rates
				? { duration, plan, power, rates, targetTime, currency, smartCostType }
				: null;
		},
	},
	watch: {
		plans(newPlans, oldPlans) {
			if (!deepEqual(newPlans, oldPlans) && newPlans.length > 0) {
				this.fetchPlanDebounced();
			}
		},
	},
	mounted() {
		if (this.plans.length > 0) {
			this.fetchPlanDebounced();
		}
	},
	methods: {
		fetchActivePlan: async function () {
			return await api.get(`loadpoints/${this.id}/plan`);
		},
		fetchPlanPreviewSoc: async function (soc, time) {
			const timeISO = time.toISOString();
			return await api.get(`loadpoints/${this.id}/plan/preview/soc/${soc}/${timeISO}`);
		},
		fetchPlanPreviewEnergy: async function (energy, time) {
			const timeISO = time.toISOString();
			return await api.get(`loadpoints/${this.id}/plan/preview/energy/${energy}/${timeISO}`);
		},
		fetchPlan: async function (preview) {
			try {
				let planRes = null;
				if (preview && this.socBasedPlanning) {
					planRes = await this.fetchPlanPreviewSoc(preview.soc, preview.time);
					this.isPreview = true;
				} else if (preview && !this.socBasedPlanning) {
					planRes = await this.fetchPlanPreviewEnergy(preview.energy, preview.time);
					this.isPreview = true;
				} else {
					planRes = await this.fetchActivePlan();
					this.isPreview = false;
				}
				this.plan = planRes.data.result;

				const tariffRes = await api.get(`tariff/planner`, {
					validateStatus: function (status) {
						return status >= 200 && status < 500;
					},
				});
				this.tariff = tariffRes.status === 404 ? { rates: [] } : tariffRes.data.result;
			} catch (e) {
				console.error(e);
			}
		},
		fetchPlanDebounced: async function (preview) {
			if (!this.debounceTimer) {
				await this.fetchPlan(preview);
				return;
			}
			clearTimeout(this.debounceTimer);
			this.debounceTimer = setTimeout(async () => await this.fetchPlan(preview), 1000);
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
				energy: this.capacity || 10,
			});
		},
		removePlan: function (index) {
			this.$emit("plan-removed", index);
		},
		updatePlan: function (data) {
			this.$emit("plan-updated", data);
		},
		previewPlan: function (data) {
			this.fetchPlanDebounced(data);
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
