<template>
	<div>
		<div v-if="recommendation" class="d-flex justify-content-between gap-3 py-1">
			<span class="d-flex align-items-center gap-2 fw-bold">
				<Optimizer />
				{{ $t("battery.optimizer.suggestion") }}
			</span>
			<span class="text-muted text-end">{{ recommendation }}</span>
		</div>
		<div v-if="forecast?.highest" class="d-flex justify-content-between gap-3 py-1">
			<span class="d-flex align-items-center gap-2 fw-bold">
				<BatteryIcon :soc="95" />
				{{ pointLabel(forecast.highest, true) }}
			</span>
			<span class="text-muted text-end">{{
				fmtDayTime(new Date(forecast.highest.time))
			}}</span>
		</div>
		<div v-if="forecast?.lowest" class="d-flex justify-content-between gap-3 py-1">
			<span class="d-flex align-items-center gap-2 fw-bold">
				<BatteryIcon :soc="10" />
				{{ pointLabel(forecast.lowest, false) }}
			</span>
			<span class="text-muted text-end">{{
				fmtDayTime(new Date(forecast.lowest.time))
			}}</span>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import type { BatteryForecast, BatteryForecastPoint } from "@/types/evcc";
import BatteryIcon from "../Energyflow/BatteryIcon.vue";
import Optimizer from "../MaterialIcon/Optimizer.vue";
import type { BatterySuggestion } from "./types";

// Optimizer rows: suggestion (not yet wired) plus the site battery high/low
// soc forecast. Simple bold-label / muted-value layout.
export default defineComponent({
	name: "OptimizerInfo",
	components: { BatteryIcon, Optimizer },
	mixins: [formatter],
	props: {
		suggestion: { type: Object as PropType<BatterySuggestion | null>, default: null },
		forecast: { type: Object as PropType<BatteryForecast | null>, default: null },
	},
	computed: {
		recommendation(): string | null {
			const s = this.suggestion;
			if (!s) return null;
			return this.$t(`battery.optimizer.action.${s.action}`);
		},
	},
	methods: {
		pointLabel(point: BatteryForecastPoint, high: boolean): string {
			const value = point.limit
				? this.$t(high ? "battery.optimizer.full" : "battery.optimizer.empty")
				: this.fmtPercentage(point.soc, 0);
			return this.$t(high ? "battery.optimizer.highest" : "battery.optimizer.lowest", {
				value,
			});
		},
	},
});
</script>
