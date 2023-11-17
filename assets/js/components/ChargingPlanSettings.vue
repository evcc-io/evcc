<template>
	<div class="mt-4">
		<div class="form-group d-lg-flex align-items-baseline mb-2 justify-content-between">
			<div v-if="plans.length > 0" class="container px-0">
				<ChargingPlanSettingsEntry
					v-for="(p, index) in plans"
					:id="`${id}_${index}`"
					:key="index"
					class="my-3"
					v-bind="p"
					:vehicle-capacity="vehicleCapacity"
					:range-per-soc="rangePerSoc"
					:soc-per-kwh="socPerKwh"
					:soc-based-charging="socBasedCharging"
					@plan-updated="(data) => updatePlan({ index, ...data })"
					@plan-removed="() => removePlan(index)"
				/>
			</div>
			<div v-else>
				<p>
					No charging target set. Set your departure and charge goals; evcc computes the
					optimal charging schedule.
				</p>
				<button class="btn btn-outline-primary" type="button" @click="addPlan">
					Set charging target
				</button>
			</div>
		</div>
		<div v-if="plans.length > 0">
			<hr />
			<h5>PREVIEW</h5>
			<!--
			<p class="mb-0">
				<span v-if="timeInThePast" class="d-block text-danger mb-1">
					{{ $t("main.targetCharge.targetIsInThePast") }}
				</span>
				<span v-if="!socBasedCharging && !targetEnergy" class="d-block text-danger mb-1">
					{{ $t("main.targetCharge.targetEnergyRequired") }}
				</span>
				<span v-if="timeTooFarInTheFuture" class="d-block text-secondary mb-1">
					{{ $t("main.targetCharge.targetIsTooFarInTheFuture") }}
				</span>
				<span v-if="costLimitExists" class="d-block text-secondary mb-1">
					{{
						$t("main.targetCharge.costLimitIgnore", {
							limit: costLimitText,
						})
					}}
				</span>
				<span v-if="['off', 'now'].includes(mode)" class="d-block text-secondary mb-1">
					{{ $t("main.targetCharge.onlyInPvMode") }}
				</span>
				&nbsp;
			</p>
			-->
			<ChargingPlanPreview
				v-if="chargingPlanPreviewProps"
				v-bind="chargingPlanPreviewProps"
			/>
		</div>
	</div>
</template>

<script>
import { CO2_TYPE } from "../units";
import ChargingPlanPreview from "./ChargingPlanPreview.vue";
import ChargingPlanSettingsEntry from "./ChargingPlanSettingsEntry.vue";
import formatter from "../mixins/formatter";
import api from "../api";

const DEFAULT_TARGET_TIME = "7:00";
const LAST_TARGET_TIME_KEY = "last_target_time";

export default {
	name: "ChargingPlanSettings",
	components: { ChargingPlanPreview, ChargingPlanSettingsEntry },
	mixins: [formatter],
	props: {
		id: [String, Number],
		plans: { type: Array, default: () => [] },
		effectiveLimitSoc: Number,
		effectivePlanTime: String,
		limitEnergy: Number,
		socBasedCharging: Boolean,
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
		timeInThePast: function () {
			const now = new Date();
			return now >= this.selectedTargetTime;
		},
		timeTooFarInTheFuture: function () {
			if (this.tariff?.rates) {
				const lastRate = this.tariff.rates[this.tariff.rates.length - 1];
				if (lastRate?.end) {
					const end = new Date(lastRate.end);
					return this.selectedTargetTime >= end;
				}
			}
			return false;
		},
		selectedTargetTime: function () {
			return new Date(this.effectivePlanTime);
		},
		targetEnergyFormatted: function () {
			return this.fmtKWh(this.targetEnergy * 1e3, true, true, 1);
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
		tariffHighest: function () {
			return this.tariff?.rates.reduce((res, slot) => {
				return Math.max(res, slot.price);
			}, 0);
		},
		costLimitExists: function () {
			return this.smartCostLimit !== 0;
		},
		costLimitText: function () {
			if (this.isCo2) {
				return this.$t("main.targetCharge.co2Limit", {
					co2: this.fmtCo2Short(this.smartCostLimit),
				});
			}
			return this.$t("main.targetCharge.priceLimit", {
				price: this.fmtPricePerKWh(this.smartCostLimit, this.currency, true),
			});
		},
		isCo2() {
			return this.smartCostType === CO2_TYPE;
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
@media (min-width: 992px) {
	.date-selection {
		width: 370px;
	}
}
.time-selection {
	flex-basis: 200px;
}
h5 {
	position: relative;
	display: inline-block;
	background-color: white;
	top: -25px;
	left: calc(50% - 50px);
	padding: 0 0.5rem;
	font-weight: normal;
	color: var(--bs-gray);
	margin-bottom: -4rem;
}
</style>
