<template>
	<p class="mb-0">
		<span v-if="['off', 'now'].includes(mode)" class="d-block text-secondary mb-1">
			{{ $t("main.targetCharge.onlyInPvMode") }}
		</span>
		<span v-if="targetIsAboveLimit" class="d-block text-danger mb-1">
			{{
				$t("main.targetCharge.targetIsAboveLimit", {
					limit: limitFmt,
					goal: goalFmt,
				})
			}}
		</span>
		<span v-if="timeInThePast" class="d-block text-danger mb-1">
			{{ $t("main.targetCharge.targetIsInThePast") }}
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
		selectedTargetTime: Date,
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
		targetIsAboveLimit: function () {
			if (this.socBasedPlanning) {
				return this.effectivePlanSoc > this.effectiveLimitSoc;
			}
			return this.limitEnergy && this.planEnergy > this.limitEnergy;
		},
		limitFmt: function () {
			if (this.socBasedPlanning) {
				return this.fmtSoc(this.effectiveLimitSoc);
			}
			return this.fmtKWh(this.limitEnergy * 1e3);
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
