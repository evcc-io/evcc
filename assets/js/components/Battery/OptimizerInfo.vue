<template>
	<div class="d-flex flex-column gap-2">
		<div v-if="recommendation" class="d-flex justify-content-between gap-3">
			<span class="d-flex align-items-center gap-2 fw-bold">
				<Optimizer :class="actionClass" />
				<router-link to="/optimize" class="evcc-default-text text-decoration-underline">
					{{ $t("battery.optimizer.suggestion") }}
				</router-link>
			</span>
			<span class="text-muted text-end">{{ recommendation }}</span>
		</div>
		<div v-if="forecast?.highest" class="d-flex justify-content-between gap-3">
			<span class="d-flex align-items-center gap-2 fw-bold">
				<BatteryIcon :soc="95" />
				{{ pointLabel(forecast.highest, true) }}
			</span>
			<span class="text-muted text-end">{{
				fmtDayTime(new Date(forecast.highest.time))
			}}</span>
		</div>
		<div v-if="forecast?.lowest" class="d-flex justify-content-between gap-3">
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
import type { BatteryForecast, BatteryForecastPoint, BatterySuggestion } from "@/types/evcc";
import BatteryIcon from "../Energyflow/BatteryIcon.vue";
import Optimizer from "../MaterialIcon/Optimizer.vue";
import { optimizerActionClass } from "@/utils/optimizer";

// Optimizer rows: suggestion plus the battery high/low soc forecast.
// Simple bold-label / muted-value layout.
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
			const action = this.suggestion?.action;
			if (!action) return null;
			return this.$t(`battery.optimizer.action.${action}`);
		},
		actionClass(): string {
			return optimizerActionClass(this.suggestion?.action);
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
