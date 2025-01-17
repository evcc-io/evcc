<template>
	<div class="mt-4">
		<div class="form-group d-lg-flex align-items-baseline justify-content-between">
			<div class="container px-0">
				<ChargingPlanStaticSettings
					:id="`lp${id}-1`"
					class="mb-2"
					v-bind="staticPlan || {}"
					:capacity="capacity"
					:range-per-soc="rangePerSoc"
					:soc-per-kwh="socPerKwh"
					:soc-based-planning="socBasedPlanning"
					:multiplePlans="multiplePlans"
					@static-plan-updated="(data) => updateStaticPlan({ index: 0, ...data })"
					@static-plan-removed="() => removeStaticPlan(0)"
					@plan-preview="previewStaticPlan"
				/>
				<div v-if="socBasedPlanning">
					<div v-if="multiplePlans" class="d-none d-lg-block">
						<hr class="mt-5" />
						<h5>
							<div class="inner mb-3" data-testid="repeating-plan-title">
								{{ $t("main.chargingPlan.repeatingPlans") }}
							</div>
						</h5>
					</div>

					<ChargingPlansRepeatingSettings
						:id="id"
						:rangePerSoc="rangePerSoc"
						:plans="repeatingPlans"
						@updated="updateRepeatingPlans"
					/>
				</div>
			</div>
		</div>
		<hr />
		<h5>
			<div class="inner" data-testid="plan-preview-title">
				<span v-if="!multiplePlans">
					{{ $t(`main.targetCharge.${noActivePlan ? "preview" : "currentPlan"}`) }}
				</span>
				<CustomSelect
					v-else-if="noActivePlan"
					:options="previewPlanOptions"
					:selected="selectedPreviewId"
					data-testid="preview-plan-select"
					@change="selectPreviewPlan($event.target.value)"
				>
					<span class="text-decoration-underline">
						{{ selectedPreviewPlanTitle }}
					</span>
				</CustomSelect>
				<span v-else>
					{{ nextPlanTitle }}
				</span>
			</div>
		</h5>
		<ChargingPlanWarnings v-bind="chargingPlanWarningsProps" />
		<ChargingPlanPreview v-bind="chargingPlanPreviewProps" />
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/plus";
import ChargingPlanPreview from "./ChargingPlanPreview.vue";
import ChargingPlanStaticSettings from "./ChargingPlanStaticSettings.vue";
import ChargingPlansRepeatingSettings from "./ChargingPlansRepeatingSettings.vue";
import ChargingPlanWarnings from "./ChargingPlanWarnings.vue";
import formatter from "../mixins/formatter";
import collector from "../mixins/collector";
import api from "../api";
import CustomSelect from "./CustomSelect.vue";
import deepEqual from "../utils/deepEqual";

const TARIFF_CACHE_TIME = 300000; // 5 minutes

export default {
	name: "ChargingPlansSettings",
	components: {
		ChargingPlanPreview,
		ChargingPlanStaticSettings,
		ChargingPlansRepeatingSettings,
		ChargingPlanWarnings,
		CustomSelect,
	},
	mixins: [formatter, collector],
	props: {
		id: [String, Number],
		staticPlan: Object,
		repeatingPlans: { type: Array, default: () => [] },
		effectiveLimitSoc: Number,
		effectivePlanTime: String,
		effectivePlanSoc: Number,
		planEnergy: Number,
		limitEnergy: Number,
		socBasedPlanning: Boolean,
		socPerKwh: Number,
		rangePerSoc: Number,
		smartCostType: String,
		currency: String,
		mode: String,
		capacity: Number,
		vehicle: Object,
		vehicleLimitSoc: Number,
		planOverrun: Number,
	},
	emits: ["static-plan-removed", "static-plan-updated", "repeating-plans-updated"],
	data: function () {
		return {
			staticPlanPreview: {},
			tariff: {},
			plan: {},
			activeTab: "time",
			debounceTimer: null,
			selectedPreviewId: 1,
			nextPlanId: 0,
		};
	},
	computed: {
		noActivePlan: function () {
			return !this.staticPlan && this.repeatingPlans.every((plan) => !plan.active);
		},
		multiplePlans: function () {
			return this.repeatingPlans.length !== 0;
		},
		selectedPreviewPlanTitle: function () {
			return this.previewPlanOptions[this.selectedPreviewId - 1]?.name;
		},
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
		previewPlanOptions: function () {
			const name = (number) => `${this.$t("main.targetCharge.preview")} #${number}`;

			// static plan
			const options = [{ value: 1, name: name(1) }];

			// repeating plans
			this.repeatingPlans.forEach((plan, index) => {
				const number = index + 2;
				options.push({
					value: number,
					name: name(number),
					disabled: !plan.weekdays.length,
				});
			});

			return options;
		},
		nextPlanTitle: function () {
			return `${this.$t("main.targetCharge.nextPlan")} #${this.nextPlanId}`;
		},
	},
	watch: {
		effectivePlanTime(newValue) {
			if (null !== newValue) {
				this.fetchPlanDebounced();
			}
		},
		staticPlan: {
			deep: true,
			handler: function (vNew, vOld) {
				if (!deepEqual(vNew, vOld)) {
					this.fetchPlanDebounced();
				}
			},
		},
		repeatingPlans: {
			deep: true,
			handler: function (vNew, vOld) {
				if (!deepEqual(vNew, vOld)) {
					this.adjustPreviewId();
					this.fetchPlanDebounced();
				}
			},
		},
	},
	mounted() {
		this.fetchPlanDebounced();
	},
	methods: {
		selectPreviewPlan: function (id) {
			this.selectedPreviewId = id;
			this.fetchPlanPreviewDebounced();
		},
		fetchPlanDebounced: async function () {
			if (this.noActivePlan) {
				await this.fetchPlanPreviewDebounced();
			} else {
				await this.fetchActivePlanDebounced();
			}
		},
		adjustPreviewId: function () {
			if (this.selectedPreviewId > this.previewPlanOptions.length) {
				this.selectedPreviewId = this.previewPlanOptions.length;
			}
		},
		fetchActivePlan: async function () {
			try {
				const res = await api.get(`loadpoints/${this.id}/plan`);
				this.plan = res.data.result;
				this.nextPlanId = this.plan.planId;
			} catch (e) {
				console.error(e);
			}
			await this.updateTariff();
		},
		fetchStaticPreviewSoc: async function (soc, time) {
			const timeISO = time.toISOString();
			return await api.get(`loadpoints/${this.id}/plan/static/preview/soc/${soc}/${timeISO}`);
		},
		fetchRepeatingPreview: async function (weekdays, soc, time, tz) {
			return await api.get(
				`loadpoints/${this.id}/plan/repeating/preview/${soc}/${weekdays}/${time}/${encodeURIComponent(tz)}`
			);
		},
		fetchStaticPreviewEnergy: async function (energy, time) {
			const timeISO = time.toISOString();
			return await api.get(
				`loadpoints/${this.id}/plan/static/preview/energy/${energy}/${timeISO}`
			);
		},
		fetchPreviewPlan: async function () {
			// only show preview of no plan is active
			if (!this.noActivePlan) return;

			try {
				let planRes = undefined;

				if (this.selectedPreviewId < 2 && this.staticPlanPreview) {
					// static plan
					const { soc, energy, time } = this.staticPlanPreview;
					if (this.socBasedPlanning) {
						planRes = await this.fetchStaticPreviewSoc(soc, new Date(time));
					} else {
						planRes = await this.fetchStaticPreviewEnergy(energy, new Date(time));
					}
				} else {
					// repeating plan
					const plan = this.repeatingPlans[this.selectedPreviewId - 2];
					if (!plan) {
						return;
					}
					const { weekdays, soc, time, tz } = plan;
					if (weekdays.length === 0) {
						return;
					}
					planRes = await this.fetchRepeatingPreview(weekdays, soc, time, tz);
				}
				this.plan = planRes.data.result;
				await this.updateTariff();
			} catch (e) {
				console.error(e);
			}
		},
		updateTariff: async function () {
			// cache tariff for 5 minutes
			if (
				this.tariff?.lastUpdate &&
				Date.now() - this.tariff.lastUpdate.getTime() <= TARIFF_CACHE_TIME
			) {
				return;
			}

			const tariffRes = await api.get(`tariff/planner`, {
				validateStatus: function (status) {
					return status >= 200 && status < 500;
				},
			});
			if (tariffRes.status === 404) {
				this.tariff = { rates: [] };
			} else {
				this.tariff = tariffRes.data.result;
				this.tariff.lastUpdate = new Date();
			}
		},
		fetchPlanPreviewDebounced: async function () {
			if (!this.debounceTimer) {
				await this.fetchPreviewPlan();
				return;
			}
			clearTimeout(this.debounceTimer);
			this.debounceTimer = setTimeout(async () => await this.fetchPreviewPlan(), 1000);
		},
		fetchActivePlanDebounced: async function () {
			if (!this.debounceTimer) {
				await this.fetchActivePlan();
				return;
			}
			clearTimeout(this.debounceTimer);
			this.debounceTimer = setTimeout(async () => await this.fetchActivePlan(), 1000);
		},
		removeStaticPlan: function (index) {
			this.$emit("static-plan-removed", index);
		},
		updateStaticPlan: function (data) {
			this.$emit("static-plan-updated", data);
		},
		updateRepeatingPlans: function (plans) {
			this.$emit("repeating-plans-updated", plans);
		},
		previewStaticPlan: function (plan) {
			this.staticPlanPreview = plan;
			this.fetchPlanPreviewDebounced();
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
