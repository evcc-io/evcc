<template>
	<Card :title="title" data-testid="battery-status-card">
		<div class="d-flex justify-content-between align-items-start">
			<LabelAndValue :label="$t('battery.card.soc')" align="start">
				<div class="d-flex align-items-center gap-2">
					<SocGauge :soc="soc" :color="color" :size="38" class="soc-gauge" />
					<span class="value fw-bold" :style="{ color }">{{ fmtPercentage(soc) }}</span>
				</div>
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
				:label="powerInfo.label"
				:value="powerInfo.value"
				align="end"
			/>
		</div>

		<OptimizerInfo
			v-if="suggestion || forecast?.highest || forecast?.lowest"
			class="optimizer"
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
import type { BatteryForecast } from "@/types/evcc";
import type { BatterySuggestion } from "./types";

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
		powerInfo(): { label: string; value: string } {
			const abs = Math.abs(this.power);
			const value = this.fmtW(abs, POWER_UNIT.AUTO);
			if (abs < 50) {
				return { label: this.$t("battery.card.power"), value };
			}
			return this.power < 0
				? { label: this.$t("battery.card.charging"), value }
				: { label: this.$t("battery.card.discharging"), value };
		},
	},
});
</script>

<style scoped>
.value {
	font-size: 18px;
}
/* let the gauge straddle the value line so the % sits level with the other values */
.soc-gauge {
	margin-top: -0.5rem;
	margin-bottom: -0.5rem;
}
.optimizer {
	margin-top: 1rem;
	padding-top: 0.5rem;
	border-top: 1px solid var(--bs-border-color);
}
</style>
