<template>
	<p class="mb-3 root" data-testid="plan-warnings">
		<span v-if="targetIsAboveLimit" class="d-block evcc-gray mb-1">
			{{ $t("main.targetCharge.targetIsAboveLimit", { limit: limitFmt }) }}
		</span>
		<span v-if="mode && ['off', 'now'].includes(mode)" class="d-block evcc-gray mb-1">
			{{ $t("main.targetCharge.onlyInPvMode") }}
		</span>
		<span v-if="timeTooFarInTheFuture" class="d-block evcc-gray mb-1">
			{{ $t("main.targetCharge.targetIsTooFarInTheFuture") }}
		</span>
		<span v-if="notReachableInTime" class="d-block text-warning mb-1">
			{{ $t("main.targetCharge.notReachableInTime", { overrun: overrunFmt }) }}
		</span>
		<span v-if="targetIsAboveVehicleLimit" class="d-block text-warning mb-1">
			{{ $t("main.targetCharge.targetIsAboveVehicleLimit") }}
		</span>
	</p>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import type { PlanWrapper } from "./types";
import type { Tariff } from "@/types/evcc";

export default defineComponent({
	name: "ChargingPlanWarnings",
	mixins: [formatter],
	props: {
		id: [String, Number],
		effectiveLimitSoc: Number,
		effectivePlanTime: String,
		effectivePlanSoc: Number,
		planEnergy: Number,
		limitEnergy: Number,
		socBasedPlanning: Boolean,
		socPerKwh: Number,
		rangePerSoc: Number,
		mode: String,
		tariff: Object as PropType<Tariff>,
		plan: Object as PropType<PlanWrapper>,
		vehicleLimitSoc: Number,
		planOverrun: Number,
	},
	computed: {
		endTime(): Date | null {
			if (!this.plan?.plan?.length) {
				return null;
			}
			const { plan } = this.plan;
			return plan[plan.length - 1].end;
		},
		overrunFmt(): string {
			if (!this.planOverrun) {
				return "";
			}
			return this.fmtDuration(this.planOverrun, true, "m");
		},
		timeTooFarInTheFuture(): boolean {
			if (!this.effectivePlanTime) {
				return false;
			}
			if (this.tariff?.rates) {
				const lastRate = this.tariff.rates[this.tariff.rates.length - 1];
				if (lastRate?.end) {
					const end = new Date(lastRate.end);
					return new Date(this.effectivePlanTime) >= end;
				}
			}
			return false;
		},
		notReachableInTime(): boolean {
			const { planTime } = this.plan || {};
			if (planTime && this.endTime) {
				const dateWanted = new Date(planTime);
				const dateEstimated = new Date(this.endTime);
				// 1 minute tolerance
				return dateEstimated.getTime() - dateWanted.getTime() > 60 * 1e3;
			}
			return false;
		},
		targetIsAboveLimit(): boolean {
			if (this.socBasedPlanning && this.effectivePlanSoc && this.effectiveLimitSoc) {
				return this.effectivePlanSoc > this.effectiveLimitSoc;
			}
			return !!this.limitEnergy && !!this.planEnergy && this.planEnergy > this.limitEnergy;
		},
		targetIsAboveVehicleLimit(): boolean {
			if (this.socBasedPlanning && this.effectivePlanSoc) {
				return this.effectivePlanSoc > (this.vehicleLimitSoc || 100);
			}
			return false;
		},
		limitFmt(): string {
			if (this.socBasedPlanning && this.effectiveLimitSoc) {
				return this.fmtSoc(this.effectiveLimitSoc);
			} else if (this.limitEnergy) {
				return this.fmtWh(this.limitEnergy * 1e3);
			} else {
				return "??";
			}
		},
	},
	methods: {
		fmtSoc(soc: number): string {
			return this.fmtPercentage(soc);
		},
	},
});
</script>

<style scoped>
.root:empty {
	display: none;
}
</style>
