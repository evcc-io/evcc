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
					@static-plan-updated="updateStaticPlan"
					@static-plan-removed="removeStaticPlan"
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
			<div class="inner d-flex align-items-center gap-1" data-testid="plan-preview-title">
				<span v-if="!multiplePlans">
					{{ $t(`main.targetCharge.${noActivePlan ? "preview" : "currentPlan"}`) }}
				</span>
				<span v-else-if="noActivePlan">{{ $t("main.targetCharge.preview") }} #1</span>
				<span v-else-if="alreadyReached">{{ $t("main.targetCharge.goalReached") }}</span>
				<span v-else>{{ nextPlanTitle }}</span>
				<button
					type="button"
					class="btn btn-sm"
					:class="strategyOpen ? 'btn-secondary' : 'evcc-gray'"
					:aria-label="$t('main.chargingPlan.strategySettings')"
					tabindex="0"
					@click="strategyOpen = !strategyOpen"
				>
					<shopicon-regular-adjust size="s"></shopicon-regular-adjust>
				</button>
			</div>
		</h5>
		<ChargingPlanStrategy
			:id="id"
			:precondition="effectivePlanStrategy?.precondition"
			:continuous="effectivePlanStrategy?.continuous"
			:disabled="strategyDisabled"
			:show="strategyOpen"
			@update="updatePlanStrategy"
		/>
		<ChargingPlanPreview v-bind="chargingPlanPreviewProps" />
		<ChargingPlanWarnings v-bind="chargingPlanWarningsProps" />
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/plus";
import Preview from "./Preview.vue";
import PlanStaticSettings from "./PlanStaticSettings.vue";
import ChargingPlanStrategy from "./PlanStrategy.vue";
import RepeatingSettings from "./PlansRepeatingSettings.vue";
import Warnings from "./Warnings.vue";
import formatter from "@/mixins/formatter";
import collector from "@/mixins/collector";
import api from "@/api";
import deepEqual from "@/utils/deepEqual";
import convertRates from "@/utils/convertRates";
import { debounceLeading } from "@/utils/debounceLeading";
import { defineComponent, type PropType } from "vue";
import type { Vehicle, CURRENCY, Forecast } from "@/types/evcc";
import type {
	StaticPlan,
	RepeatingPlan,
	PlanWrapper,
	StaticSocPlan,
	StaticEnergyPlan,
	PlanResponse,
	PlanStrategy,
} from "./types";

export default defineComponent({
	name: "ChargingPlansSettings",
	components: {
		ChargingPlanPreview: Preview,
		ChargingPlanStaticSettings: PlanStaticSettings,
		ChargingPlanStrategy,
		ChargingPlansRepeatingSettings: RepeatingSettings,
		ChargingPlanWarnings: Warnings,
	},
	mixins: [formatter, collector],
	props: {
		id: [String, Number],
		staticPlan: Object as PropType<StaticPlan>,
		repeatingPlans: { type: Array as PropType<RepeatingPlan[]>, default: () => [] },
		effectiveLimitSoc: Number,
		effectivePlanTime: String,
		effectivePlanSoc: Number,
		effectivePlanStrategy: Object as PropType<PlanStrategy>,
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
	emits: [
		"static-plan-removed",
		"static-plan-updated",
		"repeating-plans-updated",
		"plan-strategy-updated",
	],
	data() {
		return {
			staticPlanPreview: {} as StaticPlan,
			plan: {} as PlanWrapper,
			activeTab: "time",
			nextPlanId: 0,
			strategyOpen: false,
			updatePlanPreviewDebounced: null as any as () => void,
			updateActivePlanDebounced: null as any as () => void,
		};
	},
	computed: {
		noActivePlan(): boolean {
			return !this.staticPlan && this.repeatingPlans.every((plan) => !plan.active);
		},
		multiplePlans(): boolean {
			return this.repeatingPlans.length !== 0;
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
		alreadyReached(): boolean {
			return this.plan.duration === 0;
		},
		nextPlanTitle(): string {
			return `${this.$t("main.targetCharge.nextPlan")} #${this.nextPlanId}`;
		},
		strategyDisabled(): boolean {
			// options only make sense if there are variable prices
			// TODO: make this logic more robust (api fails, missing data)
			const slots = this.forecast?.planner || [];
			const values = new Set(slots.map(({ value }) => value));
			return values.size <= 1;
		},
	},
	watch: {
		effectivePlanTime(newValue: string) {
			if (null !== newValue) {
				this.updatePlanDebounced();
			}
		},
		effectivePlanStrategy: {
			deep: true,
			handler(vNew: PlanStrategy, vOld: PlanStrategy) {
				if (!deepEqual(vNew, vOld)) {
					this.updatePlanDebounced();
				}
			},
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
					this.updatePlanDebounced();
				}
			},
		},
	},
	mounted(): void {
		this.updatePlanPreviewDebounced = debounceLeading(
			async () => await this.updatePreviewPlan(),
			300
		);
		this.updateActivePlanDebounced = debounceLeading(
			async () => await this.updateActivePlan(),
			300
		);
		this.updatePlanDebounced();
	},
	methods: {
		updatePlanDebounced(): void {
			if (this.noActivePlan) {
				this.updatePlanPreviewDebounced();
			} else {
				this.updateActivePlanDebounced();
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
			const params: Record<string, unknown> = {};
			return await this.apiFetchPlan(
				`loadpoints/${this.id}/plan/static/preview/soc/${plan.soc}/${timeISO}`,
				params
			);
		},
		async fetchStaticPreviewEnergy(plan: StaticEnergyPlan): Promise<PlanResponse | undefined> {
			const timeISO = plan.time.toISOString();
			const params: Record<string, unknown> = {};
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
				if (this.staticPlanPreview) {
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
				}
				this.plan = planRes?.data ?? ({} as PlanWrapper);
			} catch (e) {
				console.error(e);
			}
		},
		removeStaticPlan(): void {
			this.$emit("static-plan-removed");
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
		updatePlanStrategy(strategy: PlanStrategy): void {
			this.$emit("plan-strategy-updated", strategy);
		},
	},
});
</script>

<style scoped>
h5 {
	position: relative;
	display: flex;
	top: -33px;
	margin-bottom: -0.5rem;
	padding: 0 0.5rem;
	justify-content: center;
}
h5 .inner {
	padding: 0 1rem;
	background-color: var(--evcc-box);
	font-weight: normal;
	color: var(--evcc-gray);
	text-transform: uppercase;
	text-align: center;
}
</style>
