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
					:multiple-plans="multiplePlans"
					:show-precondition="showPrecondition"
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
						:show-precondition="showPrecondition"
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
				<span v-else-if="alreadyReached">
					{{ $t("main.targetCharge.goalReached") }}
				</span>
				<span v-else>
					{{ nextPlanTitle }}
				</span>
			</div>
		</h5>
		<ChargingPlanPreview v-bind="chargingPlanPreviewProps" />
		<ChargingPlanWarnings v-bind="chargingPlanWarningsProps" />
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/plus";
import Preview from "./Preview.vue";
import PlanStaticSettings from "./PlanStaticSettings.vue";
import RepeatingSettings from "./PlansRepeatingSettings.vue";
import Warnings from "./Warnings.vue";
import formatter from "@/mixins/formatter";
import collector from "@/mixins/collector";
import api from "@/api";
import CustomSelect from "../Helper/CustomSelect.vue";
import deepEqual from "@/utils/deepEqual";
import convertRates from "@/utils/convertRates";
import { defineComponent, type PropType } from "vue";
import type { Vehicle, PartialBy, Timeout, SelectOption, CURRENCY, Forecast } from "@/types/evcc";
import type {
	StaticPlan,
	RepeatingPlan,
	PlanWrapper,
	StaticSocPlan,
	StaticEnergyPlan,
	PlanResponse,
} from "./types";

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
		forecast: Object as PropType<Forecast>,
	},
	emits: ["static-plan-removed", "static-plan-updated", "repeating-plans-updated"],
	data() {
		return {
			staticPlanPreview: {} as StaticPlan,
			plan: {} as PlanWrapper,
			activeTab: "time",
			debounceTimer: null as Timeout,
			selectedPreviewId: 1,
			nextPlanId: 0,
		};
	},
	computed: {
		noActivePlan(): boolean {
			return !this.staticPlan && this.repeatingPlans.every((plan) => !plan.active);
		},
		multiplePlans(): boolean {
			return this.repeatingPlans.length !== 0;
		},
		selectedPreviewPlanTitle(): string {
			return this.previewPlanOptions[this.selectedPreviewId - 1]?.name;
		},
		chargingPlanWarningsProps(): any {
			return this.collectProps(Warnings);
		},
		chargingPlanPreviewProps(): any {
			const forecastSlots = this.forecast?.planner || [];
			const rates = convertRates(forecastSlots);
			const { duration, plan, power, planTime } = this.plan;
			const targetTime = planTime ? new Date(planTime) : null;
			const { currency, smartCostType } = this;
			return rates
				? { duration, plan, power, rates, targetTime, currency, smartCostType }
				: null;
		},
		previewPlanOptions(): SelectOption<number>[] {
			const name = (n: number) => `${this.$t("main.targetCharge.preview")} #${n}`;

			// static plan
			const options = [{ value: 1, name: name(1) }] as SelectOption<number>[];

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
		alreadyReached(): boolean {
			return this.plan.duration === 0;
		},
		nextPlanTitle(): string {
			return `${this.$t("main.targetCharge.nextPlan")} #${this.nextPlanId}`;
		},
		showPrecondition(): boolean {
			// only show option if planner forecast has different values
			const slots = this.forecast?.planner || [];
			const values = new Set(slots.map(({ value }) => value));
			return values.size > 1;
		},
	},
	watch: {
		effectivePlanTime(newValue: string) {
			if (null !== newValue) {
				this.updatePlanDebounced();
			}
		},
		staticPlan: {
			deep: true,
			handler(vNew: StaticPlan, vOld: StaticPlan) {
				if (!deepEqual(vNew, vOld)) {
					this.updatePlanDebounced();
				}
			},
		},
		repeatingPlans: {
			deep: true,
			handler(vNew: RepeatingPlan[], vOld: RepeatingPlan[]) {
				if (!deepEqual(vNew, vOld)) {
					this.adjustPreviewId();
					this.updatePlanDebounced();
				}
			},
		},
	},
	mounted(): void {
		this.updatePlanDebounced();
	},
	methods: {
		selectPreviewPlan(id: number): void {
			this.selectedPreviewId = id;
			this.updatePlanDebounced();
		},
		async updatePlanDebounced() {
			if (this.noActivePlan) {
				await this.updatePlanPreviewDebounced();
			} else {
				await this.updateActivePlanDebounced();
			}
		},
		adjustPreviewId(): void {
			if (this.selectedPreviewId > this.previewPlanOptions.length) {
				this.selectedPreviewId = this.previewPlanOptions.length;
			}
		},
		async updateActivePlan(): Promise<void> {
			try {
				const res = await this.apiFetchPlan(`loadpoints/${this.id}/plan`);
				this.plan = res?.data ?? ({} as PlanWrapper);
				this.nextPlanId = this.plan.planId;
			} catch (e) {
				console.error(e);
			}
		},
		async fetchStaticPreviewSoc(plan: StaticSocPlan): Promise<PlanResponse | undefined> {
			const timeISO = plan.time.toISOString();
			const params = plan.precondition ? { precondition: plan.precondition } : undefined;
			return await this.apiFetchPlan(
				`loadpoints/${this.id}/plan/static/preview/soc/${plan.soc}/${timeISO}`,
				params
			);
		},
		async fetchRepeatingPreview(
			plan: PartialBy<RepeatingPlan, "active">
		): Promise<PlanResponse | undefined> {
			return await this.apiFetchPlan(
				`loadpoints/${this.id}/plan/repeating/preview/${plan.soc}/${plan.weekdays}/${plan.time}/${encodeURIComponent(plan.tz)}`
			);
		},
		async fetchStaticPreviewEnergy(plan: StaticEnergyPlan): Promise<PlanResponse | undefined> {
			const timeISO = plan.time.toISOString();
			const params = plan.precondition ? { precondition: plan.precondition } : undefined;
			return await this.apiFetchPlan(
				`loadpoints/${this.id}/plan/static/preview/energy/${plan.energy}/${timeISO}`,
				params
			);
		},
		async apiFetchPlan(
			url: string,
			params?: Record<string, unknown>
		): Promise<PlanResponse | undefined> {
			try {
				const res = (await api.get(url, {
					validateStatus: (code) => [200, 404].includes(code),
					params,
				})) as PlanResponse;
				if (res.status === 404) {
					return { data: {} } as PlanResponse;
				}
				return res;
			} catch (e) {
				console.error(e);
				return;
			}
		},
		async updatePreviewPlan(): Promise<void> {
			// only show preview if no plan is active
			if (!this.noActivePlan) return;

			try {
				let planRes: PlanResponse | undefined = undefined;

				if (this.selectedPreviewId < 2 && this.staticPlanPreview) {
					// static plan
					let plan = this.staticPlanPreview;
					if (this.socBasedPlanning) {
						plan = plan as StaticSocPlan;
						planRes = await this.fetchStaticPreviewSoc({
							soc: plan.soc,
							time: plan.time,
							precondition: plan.precondition,
						});
					} else {
						plan = plan as StaticEnergyPlan;
						planRes = await this.fetchStaticPreviewEnergy({
							energy: plan.energy,
							time: plan.time,
							precondition: plan.precondition,
						});
					}
				} else {
					// repeating plan
					const plan = this.repeatingPlans[this.selectedPreviewId - 2];
					if (!plan) {
						return;
					}
					const { weekdays, soc, time, tz, precondition } = plan;
					if (weekdays.length === 0) {
						return;
					}
					planRes = await this.fetchRepeatingPreview({
						weekdays,
						soc,
						time,
						tz,
						precondition,
					});
				}
				this.plan = planRes?.data ?? ({} as PlanWrapper);
			} catch (e) {
				console.error(e);
			}
		},
		async updatePlanPreviewDebounced(): Promise<void> {
			if (!this.debounceTimer) {
				await this.updatePreviewPlan();
				return;
			}
			clearTimeout(this.debounceTimer);
			this.debounceTimer = setTimeout(async () => await this.updatePreviewPlan(), 1000);
		},
		async updateActivePlanDebounced(): Promise<void> {
			if (!this.debounceTimer) {
				await this.updateActivePlan();
				return;
			}
			clearTimeout(this.debounceTimer);
			this.debounceTimer = setTimeout(async () => await this.updateActivePlan(), 1000);
		},
		removeStaticPlan(index: number): void {
			this.$emit("static-plan-removed", index);
		},
		updateStaticPlan(plan: StaticPlan): void {
			this.$emit("static-plan-updated", plan);
		},
		updateRepeatingPlans(plans: RepeatingPlan[]): void {
			this.$emit("repeating-plans-updated", plans);
		},
		previewStaticPlan(plan: StaticPlan): void {
			this.staticPlanPreview = plan;
			this.updatePlanPreviewDebounced();
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
