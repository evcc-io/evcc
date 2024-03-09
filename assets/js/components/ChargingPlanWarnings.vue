<template>
	<p class="mb-0" data-testid="plan-warnings">
		<span v-if="targetIsAboveLimit" class="d-block text-secondary mb-1">
			{{ $t("main.targetCharge.targetIsAboveLimit", { limit: limitFmt }) }}
		</span>
		<span v-if="['off', 'now'].includes(mode)" class="d-block text-secondary mb-1">
			{{ $t("main.targetCharge.onlyInPvMode") }}
		</span>
		<span v-if="timeTooFarInTheFuture" class="d-block text-secondary mb-1">
			{{ $t("main.targetCharge.targetIsTooFarInTheFuture") }}
		</span>
		<span v-if="costLimitExists" class="d-block text-secondary mb-1">
			{{ $t("main.targetCharge.costLimitIgnore", { limit: costLimitText }) }}
		</span>
		<span v-if="notReachableInTime" class="d-block text-warning mb-1">
			{{ $t("main.targetCharge.notReachableInTime", { endTime: endTimeFmt }) }}
		</span>
		<span v-if="targetIsAboveVehicleLimit" class="d-block text-danger mb-1">
			{{ $t("main.targetCharge.targetIsAboveVehicleLimit", { limit: vehicleLimitFmt }) }}
		</span>
	</p>
</template>

<script>
import { CO2_TYPE } from "../units";
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
		smartCostLimit: Number,
		smartCostType: String,
		currency: String,
		mode: String,
		tariff: Object,
		plan: Object,
		vehicleTargetSoc: Number,
	},
	computed: {
		endTime: function () {
			if (!this.plan?.plan?.length) {
				return null;
			}
			const { plan } = this.plan;
			return plan[plan.length - 1].end;
		},
		endTimeFmt: function () {
			if (!this.endTime) {
				return "";
			}
			return this.fmtAbsoluteDate(new Date(this.endTime));
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
				// account for rounding errors
				dateWanted.setSeconds(60);
				dateEstimated.setSeconds(0);
				return dateWanted < dateEstimated;
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
				return this.effectivePlanSoc > (this.vehicleTargetSoc || 100);
			}
			return false;
		},
		limitFmt: function () {
			if (this.socBasedPlanning) {
				return this.fmtSoc(this.effectiveLimitSoc);
			}
			return this.fmtKWh(this.limitEnergy * 1e3);
		},
		vehicleLimitFmt: function () {
			return this.fmtSoc(this.vehicleTargetSoc);
		},
		goalFmt: function () {
			if (this.socBasedPlanning) {
				return this.fmtSoc(this.effectivePlanSoc);
			}
			return this.fmtKWh(this.planEnergy * 1e3);
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
	methods: {
		fmtSoc(soc) {
			return `${Math.round(soc)}%`;
		},
	},
};
</script>
