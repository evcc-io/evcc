<template>
	<Card data-testid="battery-status-card">
		<div class="values">
			<LabelAndValue :label="title" :color="color" align="start">
				<h3 class="value m-0 fw-bold d-flex align-items-center gap-2" :style="{ color }">
					<SocGauge
						:soc="soc"
						:color="color"
						:power="power"
						:mode="controllable ? batteryMode : undefined"
					/>
					{{ fmtPercentage(soc) }}
				</h3>
			</LabelAndValue>

			<LabelAndValue
				class="text-center"
				:label="$t('battery.card.energy')"
				:value="hasCapacity ? energyKWh : '–'"
				:valueFmt="hasCapacity ? energyFmt : undefined"
				:extraValue="hasCapacity ? energySub : undefined"
				align="center"
			/>

			<LabelAndValue
				class="text-end"
				:label="$t('battery.card.power')"
				:value="powerValue"
				:extraValue="powerState"
				align="end"
			/>
		</div>

		<OptimizerInfo
			v-if="suggestion || forecast?.highest || forecast?.lowest"
			class="mt-3 pt-3 border-top"
			:suggestion="suggestion"
			:forecast="forecast"
		/>
	</Card>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter, { POWER_UNIT } from "@/mixins/formatter";
import Card from "../Helper/Card.vue";
import LabelAndValue from "../Helper/LabelAndValue.vue";
import SocGauge from "./SocGauge.vue";
import OptimizerInfo from "./OptimizerInfo.vue";
import { BATTERY_MODE, type BatteryForecast, type BatterySuggestion } from "@/types/evcc";

export default defineComponent({
	name: "BatteryStatusCard",
	components: { Card, LabelAndValue, SocGauge, OptimizerInfo },
	mixins: [formatter],
	inheritAttrs: false,
	props: {
		title: { type: String, default: "" },
		soc: { type: Number, default: 0 },
		power: { type: Number, default: 0 }, // W, + discharging / - charging
		capacity: { type: Number, default: 0 }, // kWh, 0 = unspecified
		color: { type: String, default: "" },
		controllable: { type: Boolean, default: false },
		batteryMode: String as PropType<BATTERY_MODE>,
		suggestion: { type: Object as PropType<BatterySuggestion | null>, default: null },
		forecast: { type: Object as PropType<BatteryForecast | null>, default: null },
	},
	computed: {
		hasCapacity(): boolean {
			return this.capacity > 0;
		},
		energyKWh(): number {
			return (this.soc / 100) * this.capacity;
		},
		energyFmt(): (n: number) => string {
			return (n: number) => this.fmtWh(n * 1e3, POWER_UNIT.KW, true, 1);
		},
		energySub(): string {
			return this.$t("battery.card.ofTotal", {
				total: this.fmtWh(this.capacity * 1e3, POWER_UNIT.KW, true, 1),
			});
		},
		powerValue(): string {
			return this.fmtW(Math.abs(this.power), POWER_UNIT.AUTO);
		},
		// mode takes priority over the raw power reading when the battery is controllable
		statusState():
			| BATTERY_MODE.HOLD
			| BATTERY_MODE.HOLDCHARGE
			| BATTERY_MODE.CHARGE
			| "charging"
			| "discharging"
			| "idle" {
			if (this.controllable) {
				if (this.batteryMode === BATTERY_MODE.HOLD) return BATTERY_MODE.HOLD;
				if (this.batteryMode === BATTERY_MODE.HOLDCHARGE) return BATTERY_MODE.HOLDCHARGE;
				if (this.batteryMode === BATTERY_MODE.CHARGE) return BATTERY_MODE.CHARGE;
			}
			const abs = Math.abs(this.power);
			if (abs < 50) return "idle";
			return this.power < 0 ? "charging" : "discharging";
		},
		powerState(): string | undefined {
			switch (this.statusState) {
				case BATTERY_MODE.HOLD:
					return this.$t("battery.card.hold");
				case BATTERY_MODE.HOLDCHARGE:
					return this.$t("battery.card.holdCharge");
				case BATTERY_MODE.CHARGE:
					return this.$t("battery.card.gridCharge");
				case "charging":
					return this.$t("battery.card.charging");
				case "discharging":
					return this.$t("battery.card.discharging");
				default:
					return undefined;
			}
		},
	},
});
</script>

<style scoped>
.value {
	font-size: 18px;
}
.values {
	display: grid;
	grid-template-columns: repeat(3, 1fr);
	align-items: start;
	gap: 0.5rem;
}
.optimizer {
	margin-top: 1rem;
	padding-top: 1rem;
	border-top: 1px solid var(--bs-border-color);
}
.values :deep(.root) {
	margin-bottom: 0;
	min-width: 0;
}
</style>
