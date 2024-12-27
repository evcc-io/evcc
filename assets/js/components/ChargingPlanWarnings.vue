<template>
	<p class="mb-3 root" data-testid="plan-warnings">
		<span v-if="targetIsAboveLimit" class="d-block evcc-gray mb-1">
			{{ $t("main.targetCharge.targetIsAboveLimit", { limit: limitFmt }) }}
		</span>
		<span v-if="['off', 'now'].includes(mode)" class="d-block evcc-gray mb-1">
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

<script>
import formatter from "../mixins/formatter";

export default {
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
		tariff: Object,
		plan: Object,
		vehicleLimitSoc: Number,
		planOverrun: Number,
	},
	computed: {
		endTime: function () {
			if (!this.plan?.plan?.length) {
				return null;
			}
			const { plan } = this.plan;
			return plan[plan.length - 1].end;
		},
		overrunFmt: function () {
			if (!this.planOverrun) {
				return "";
			}
			return this.fmtDuration(this.planOverrun, true, "m");
		},
		timeTooFarInTheFuture: function () {
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
		notReachableInTime: function () {
			const { planTime } = this.plan || {};
			if (planTime && this.endTime) {
				const dateWanted = new Date(planTime);
				const dateEstimated = new Date(this.endTime);
				// 1 minute tolerance
				return dateEstimated - dateWanted > 60 * 1e3;
			}
			return false;
		},
		targetIsAboveLimit: function () {
			if (this.socBasedPlanning) {
				return this.effectivePlanSoc > this.effectiveLimitSoc;
			}
			return this.limitEnergy && this.planEnergy > this.limitEnergy;
		},
		targetIsAboveVehicleLimit: function () {
			if (this.socBasedPlanning) {
				return this.effectivePlanSoc > (this.vehicleLimitSoc || 100);
			}
			return false;
		},
		limitFmt: function () {
			if (this.socBasedPlanning) {
				return this.fmtSoc(this.effectiveLimitSoc);
			}
			return this.fmtWh(this.limitEnergy * 1e3);
		},
		goalFmt: function () {
			if (this.socBasedPlanning) {
				return this.fmtSoc(this.effectivePlanSoc);
			}
			return this.fmtWh(this.planEnergy * 1e3);
		},
	},
	methods: {
		fmtSoc(soc) {
			return this.fmtPercentage(soc);
		},
	},
};
</script>

<style scoped>
.root:empty {
	display: none;
}
</style>
