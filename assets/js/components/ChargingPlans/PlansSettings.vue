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

<script lang="ts">
import "@h2d2/shopicons/es/regular/plus";
import Preview from "./Preview.vue";
import PlanStaticSettings from "./PlanStaticSettings.vue";
import RepeatingSettings from "./PlansRepeatingSettings.vue";
import Warnings from "./Warnings.vue";
import formatter from "../../mixins/formatter.js";
import collector from "../../mixins/collector.js";
import api from "../../api.js";
import CustomSelect from "../Helper/CustomSelect.vue";
import deepEqual from "../../utils/deepEqual.js";
import { defineComponent, type PropType } from "vue";
import type {
	StaticPlan,
	RepeatingPlan,
	Vehicle,
	StaticEnergyPlan,
	PartialBy,
	StaticSocPlan,
	Timeout,
	Tariff,
	CustomSelectOption as SelectableOption,
	PlanWrapper,
	CURRENCY,
} from "assets/js/types/evcc.js";

const TARIFF_CACHE_TIME = 300000; // 5 minutes

export default defineComponent({
	name: "ChargingPlansSettings",
	components: {
		ChargingPlanPreview: Preview,
		ChargingPlanStaticSettings: PlanStaticSettings,
		ChargingPlansRepeatingSettings: RepeatingSettings,
		ChargingPlanWarnings: Warnings,
		CustomSelect,
	},
	mixins: [formatter, collector],
	props: {
		id: [String, Number],
		staticPlan: Object as PropType<StaticPlan>,
		repeatingPlans: { type: Array as PropType<RepeatingPlan[]>, default: () => [] },
		effectiveLimitSoc: Number,
		effectivePlanTime: String,
		effectivePlanSoc: Number,
		planEnergy: Number,
		limitEnergy: Number,
		socBasedPlanning: Boolean,
		socPerKwh: Number,
		rangePerSoc: Number,
		smartCostType: String,
		currency: String as PropType<CURRENCY>,
		mode: String,
		capacity: Number,
		vehicle: Object as PropType<Vehicle>,
		vehicleLimitSoc: Number,
		planOverrun: Number,
	},
	emits: ["static-plan-removed", "static-plan-updated", "repeating-plans-updated"],
	data() {
		return {
			staticPlanPreview: {} as StaticPlan,
			tariff: {} as Tariff,
			plan: {} as PlanWrapper,
			activeTab: "time",
			debounceTimer: null as Timeout,
			selectedPreviewId: 1,
			nextPlanId: 0,
		};
	},
	computed: {
		noActivePlan() {
			return !this.staticPlan && this.repeatingPlans.every((plan) => !plan.active);
		},
		multiplePlans() {
			return this.repeatingPlans.length !== 0;
		},
		selectedPreviewPlanTitle() {
			return this.previewPlanOptions[this.selectedPreviewId - 1]?.name;
		},
		chargingPlanWarningsProps() {
			return this.collectProps(Warnings);
		},
		chargingPlanPreviewProps() {
			const { rates } = this.tariff;
			const { duration, plan, power, planTime } = this.plan;
			const targetTime = planTime ? new Date(planTime) : null;
			const { currency, smartCostType } = this;
			return rates
				? { duration, plan, power, rates, targetTime, currency, smartCostType }
				: null;
		},
		previewPlanOptions() {
			const name = (n: number) => `${this.$t("main.targetCharge.preview")} #${n}`;

			// static plan
			const options = [{ value: 1, name: name(1) }] as SelectableOption[];

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
		nextPlanTitle() {
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
			handler(vNew, vOld) {
				if (!deepEqual(vNew, vOld)) {
					this.fetchPlanDebounced();
				}
			},
		},
		repeatingPlans: {
			deep: true,
			handler(vNew, vOld) {
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
		selectPreviewPlan(id: number) {
			this.selectedPreviewId = id;
			this.fetchPlanPreviewDebounced();
		},
		async fetchPlanDebounced() {
			if (this.noActivePlan) {
				await this.fetchPlanPreviewDebounced();
			} else {
				await this.fetchActivePlanDebounced();
			}
		},
		adjustPreviewId() {
			if (this.selectedPreviewId > this.previewPlanOptions.length) {
				this.selectedPreviewId = this.previewPlanOptions.length;
			}
		},
		async fetchActivePlan() {
			try {
				const res = await this.apiFetchPlan(`loadpoints/${this.id}/plan`);
				this.plan = res?.data.result;
				this.nextPlanId = this.plan.planId;
			} catch (e) {
				console.error(e);
			}
			await this.updateTariff();
		},
		async fetchStaticPreviewSoc(plan: StaticSocPlan) {
			const timeISO = plan.time.toISOString();
			return await this.apiFetchPlan(
				`loadpoints/${this.id}/plan/static/preview/soc/${plan.soc}/${timeISO}`
			);
		},
		async fetchRepeatingPreview(plan: PartialBy<RepeatingPlan, "active">) {
			return await this.apiFetchPlan(
				`loadpoints/${this.id}/plan/repeating/preview/${plan.soc}/${plan.weekdays}/${plan.time}/${encodeURIComponent(plan.tz)}`
			);
		},
		async fetchStaticPreviewEnergy(plan: StaticEnergyPlan) {
			const timeISO = plan.time.toISOString();
			return await this.apiFetchPlan(
				`loadpoints/${this.id}/plan/static/preview/energy/${plan.energy}/${timeISO}`
			);
		},
		async apiFetchPlan(url: string) {
			try {
				const res = await api.get(url, {
					validateStatus: (code) => [200, 404].includes(code),
				});
				if (res.status === 404) {
					return { data: { result: {} } };
				}
				return res;
			} catch (e) {
				console.error(e);
				return;
			}
		},
		async fetchPreviewPlan() {
			// only show preview of no plan is active
			if (!this.noActivePlan) return;

			try {
				let planRes = undefined;

				if (this.selectedPreviewId < 2 && this.staticPlanPreview) {
					// static plan
					let plan = this.staticPlanPreview;
					if (this.socBasedPlanning) {
						plan = plan as StaticSocPlan;
						planRes = await this.fetchStaticPreviewSoc({
							soc: plan.soc,
							time: plan.time,
						});
					} else {
						plan = plan as StaticEnergyPlan;
						planRes = await this.fetchStaticPreviewEnergy({
							energy: plan.energy,
							time: plan.time,
						});
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
					planRes = await this.fetchRepeatingPreview({ weekdays, soc, time, tz });
				}
				this.plan = planRes?.data.result;
				await this.updateTariff();
			} catch (e) {
				console.error(e);
			}
		},
		async updateTariff() {
			// cache tariff for 5 minutes
			if (
				this.tariff?.lastUpdate &&
				Date.now() - this.tariff.lastUpdate.getTime() <= TARIFF_CACHE_TIME
			) {
				return;
			}

			const tariffRes = await api.get(`tariff/planner`, {
				validateStatus(status) {
					return status >= 200 && status < 500;
				},
			});

			this.tariff = {
				rates: tariffRes.status === 404 ? [] : tariffRes.data.result,
				lastUpdate: new Date(),
			};
		},
		async fetchPlanPreviewDebounced() {
			if (!this.debounceTimer) {
				await this.fetchPreviewPlan();
				return;
			}
			clearTimeout(this.debounceTimer);
			this.debounceTimer = setTimeout(async () => await this.fetchPreviewPlan(), 1000);
		},
		async fetchActivePlanDebounced() {
			if (!this.debounceTimer) {
				await this.fetchActivePlan();
				return;
			}
			clearTimeout(this.debounceTimer);
			this.debounceTimer = setTimeout(async () => await this.fetchActivePlan(), 1000);
		},
		removeStaticPlan(index: number) {
			this.$emit("static-plan-removed", index);
		},
		updateStaticPlan(plan: StaticPlan) {
			this.$emit("static-plan-updated", plan);
		},
		updateRepeatingPlans(plans: RepeatingPlan[]) {
			this.$emit("repeating-plans-updated", plans);
		},
		previewStaticPlan(plan: StaticPlan) {
			this.staticPlanPreview = plan;
			this.fetchPlanPreviewDebounced();
		},
	},
});
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
