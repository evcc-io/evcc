<template>
	<div class="mt-4">
		<div class="form-group d-lg-flex align-items-baseline justify-content-between">
			<div class="container px-0">
				<ChargingPlanStaticSettings
					:id="`lp${id}-1`"
					class="mb-2"
					v-bind="plans[0] || {}"
					:capacity="capacity"
					:range-per-soc="rangePerSoc"
					:soc-per-kwh="socPerKwh"
					:soc-based-planning="socBasedPlanning"
					:numberPlans="numberPlans"
					@static-plan-updated="(data) => updateStaticPlan({ index: 0, ...data })"
					@static-plan-removed="() => removeStaticPlan(0)"
					@plan-preview="previewStaticPlan"
				/>
				<div v-if="socBasedPlanning">
					<div v-if="numberPlans" class="d-none d-lg-block">
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
						:initialPlans="repeatingPlans"
						:numberPlans="numberPlans"
						@repeating-plans-updated="updateRepeatingPlans"
						@plans-preview="previewRepeatingPlans"
					/>
				</div>
			</div>
		</div>
		<hr />
		<h5>
			<div class="inner">
				<CustomSelect
					v-if="showPreviewOptions"
					:options="previewPlanOptions"
					:selected="selectedPreviewPlanTitle"
					@change="changePreviewPlan"
				>
					<span data-testid="plan-preview-title" class="text-decoration-underline">
						{{ selectedPreviewPlanTitle }}
					</span>
				</CustomSelect>
				<span v-else data-testid="plan-preview-title">
					{{ $t(`main.targetCharge.currentPlan`) + ` #${nextPlanId}` }}
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

const DEFAULT_TARGET_TIME = "7:00";
const LAST_TARGET_TIME_KEY = "last_target_time";

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
		plans: { type: Array, default: () => [] },
		repeatingPlans: { type: Array, default: () => [] },
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
		vehicleLimitSoc: Number,
		planOverrun: Number,
	},
	emits: ["static-plan-removed", "static-plan-updated", "repeating-plans-updated"],
	data: function () {
		return {
			tariff: {},
			plan: {},
			activeTab: "time",
			debounceTimer: null,
			// Since we want to show unapplied changes the user made in the UI we have to store these plans separately
			plansForPreview: { repeating: this.repeatingPlans, static: this.plans[0] },
			selectedPreviewPlanId: 1,
			nextPlanId: 0,
		};
	},
	computed: {
		showPreviewOptions: function () {
			return 0 === this.nextPlanId;
		},
		numberPlans: function () {
			return this.plansForPreview.repeating.length !== 0;
		},
		selectedPreviewPlanTitle: function () {
			return this.previewPlanOptions[this.selectedPreviewPlanId - 1].name;
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
			const options = [];

			if (0 !== this.nextPlanId) {
				options.push({
					value: this.nextPlanId,
					name: this.$t("main.targetCharge.nextPlan") + " #" + this.nextPlanId,
					disabled: false,
				});
			}

			// don't show the static plan twice
			if (1 !== this.nextPlanId) {
				options.push({
					value: 1,
					name: this.$t("main.targetCharge.preview") + " #1",
					disabled: false,
				});
			}

			this.plansForPreview.repeating.forEach((plan, index) => {
				if (index + 2 !== this.nextPlanId) {
					options.push({
						value: index + 2,
						name: this.$t("main.targetCharge.preview") + " #" + (index + 2),
						disabled: 0 === plan.weekdays.length,
					});
				}
			});

			return options.sort((a, b) => {
				return a.value > b.value;
			});
		},
	},
	watch: {
		effectivePlanTime() {
			this.fetchActivePlanDebounced();
		},
	},
	mounted() {
		this.fetchActivePlanDebounced();
		this.fetchPlanPreviewDebounced();
	},
	methods: {
		changePreviewPlan: function (event) {
			this.selectedPreviewPlanId = parseInt(event.target.value);
			this.fetchPlanPreviewDebounced();
		},
		fetchActivePlan: async function () {
			await api
				.get(`loadpoints/${this.id}/plan`)
				.then((response) => {
					const planID = response.data.result.planId;
					this.selectedPreviewPlanId = planID;
					this.nextPlanId = planID;
					this.fetchPlanPreviewDebounced();
				})
				.catch(function (error) {
					console.error(error);
				});
		},
		fetchStaticPlanPreview: async function (soc, time) {
			const timeISO = time.toISOString();
			return await api.get(`loadpoints/${this.id}/plan/static/preview/soc/${soc}/${timeISO}`);
		},
		fetchRepeatingPlanPreview: async function (weekdays, soc, time) {
			const timeInUTC = this.fmtDayHourMinute(time, true)[0];
			return await api.get(
				`loadpoints/${this.id}/plan/repeating/preview/${weekdays}/${timeInUTC}/${soc}`
			);
		},
		fetchPlanPreviewEnergy: async function (energy, time) {
			const timeISO = time.toISOString();
			return await api.get(
				`loadpoints/${this.id}/plan/static/preview/energy/${energy}/${timeISO}`
			);
		},
		fetchPlan: async function () {
			try {
				let planRes = undefined;
				const planIdToPreview = this.showPreviewOptions ? this.selectedPreviewPlanId : this.nextPlanId;

				if (planIdToPreview === 1) {
					const planToPreview = this.plansForPreview.static;
					planToPreview.time = new Date(planToPreview.time);

					if (this.socBasedPlanning) {
						planRes = await this.fetchStaticPlanPreview(
							planToPreview.soc,
							planToPreview.time
						);
					} else {
						planRes = await this.fetchPlanPreviewEnergy(
							planToPreview.energy,
							planToPreview.time
						);
					}
				} else {
					const planToPreview =
						this.plansForPreview.repeating[planIdToPreview - 2];

					if (0 === planToPreview.weekdays.length) {
						return;
					}

					planRes = await this.fetchRepeatingPlanPreview(
						planToPreview.weekdays,
						planToPreview.soc,
						planToPreview.time
					);
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
		fetchPlanPreviewDebounced: async function () {
			if (!this.debounceTimer) {
				await this.fetchPlan();
				return;
			}
			clearTimeout(this.debounceTimer);
			this.debounceTimer = setTimeout(async () => await this.fetchPlan(), 1000);
		},
		fetchActivePlanDebounced: async function () {
			if (!this.debounceTimer) {
				await this.fetchActivePlan();
				return;
			}
			clearTimeout(this.debounceTimer);
			this.debounceTimer = setTimeout(
				async () => await this.fetchActivePlan(),
				1000
			);
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
		addStaticPlan: function () {
			this.$emit("static-plan-updated", {
				time: this.defaultDate(),
				soc: 100,
				energy: this.capacity || 10,
			});
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
			this.plansForPreview.static = plan;

			if (this.showPreviewOptions) {
				if (1 !== this.selectedPreviewPlanId) {
					return;
				}
			} else {
				if (1 !== this.nextPlanId) {
					return;
				}
			}

			this.fetchPlanPreviewDebounced();
		},
		previewRepeatingPlans: function (plansData) {
			const { plans, index } = plansData;
			const l = this.plansForPreview.repeating.length;

			this.plansForPreview.repeating = plans;

			if (this.showPreviewOptions) {
				// a plan was removed -> adjust selected preview plan to move correctly
				if (l > plans.length) {
					if (this.selectedPreviewPlanId - 1 === l) {
						this.selectedPreviewPlanId--;
						this.fetchPlanPreviewDebounced();
					} else if (this.selectedPreviewPlanId > index + 2) {
						this.selectedPreviewPlanId--;
						this.fetchPlanPreviewDebounced();
					} else if (this.selectedPreviewPlanId === index + 2) {
						this.fetchPlanPreviewDebounced();
					}
				} else {
					if (this.selectedPreviewPlanId === index + 2) {
						this.fetchPlanPreviewDebounced();
					}
				}
			} else {
				if (index + 2 === this.nextPlanId) {
					this.fetchPlanPreviewDebounced();
				}
			}
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
