<template>
	<div v-if="suggestionValue || highest || lowest" class="d-flex flex-column gap-2">
		<div v-if="suggestion && suggestionValue" class="d-flex justify-content-between gap-3">
			<span class="d-flex align-items-center gap-2 fw-bold">
				<Optimizer />
				<router-link to="/optimize" class="evcc-default-text text-decoration-underline">
					{{ $t("battery.optimizer.suggestion") }}
				</router-link>
			</span>
			<span class="d-flex align-items-center gap-1 text-muted text-end">
				{{ suggestionValue }}
				<BatterySuggestionIcon :action="suggestion.action" />
			</span>
		</div>
		<div v-if="highest" class="d-flex justify-content-between gap-3">
			<span class="d-flex align-items-center gap-2 fw-bold">
				<BatteryIcon :soc="highest.soc" />
				{{ pointLabel(highest, true) }}
			</span>
			<span class="text-muted text-end">{{ fmtDayTime(new Date(highest.time)) }}</span>
		</div>
		<div v-if="lowest" class="d-flex justify-content-between gap-3">
			<span class="d-flex align-items-center gap-2 fw-bold">
				<BatteryIcon :soc="lowest.soc" />
				{{ pointLabel(lowest, false) }}
			</span>
			<span class="text-muted text-end">{{ fmtDayTime(new Date(lowest.time)) }}</span>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import minuteTicker from "@/mixins/minuteTicker";
import type { BatteryForecast, BatteryForecastPoint, BatterySuggestion } from "@/types/evcc";
import BatteryIcon from "../Energyflow/BatteryIcon.vue";
import Optimizer from "../MaterialIcon/Optimizer.vue";
import BatterySuggestionIcon from "./BatterySuggestionIcon.vue";

// hide forecast points whose time has passed
function upcoming(
	point: BatteryForecastPoint | null | undefined,
	now: Date
): BatteryForecastPoint | null {
	return point && new Date(point.time) > now ? point : null;
}

// Optimizer rows: suggestion plus the battery high/low soc forecast.
// Simple bold-label / muted-value layout.
export default defineComponent({
	name: "OptimizerInfo",
	components: { BatteryIcon, Optimizer, BatterySuggestionIcon },
	mixins: [formatter, minuteTicker],
	props: {
		suggestion: { type: Object as PropType<BatterySuggestion | null>, default: null },
		forecast: { type: Object as PropType<BatteryForecast | null>, default: null },
	},
	computed: {
		highest(): BatteryForecastPoint | null {
			return upcoming(this.forecast?.highest, this.everyMinute);
		},
		lowest(): BatteryForecastPoint | null {
			return upcoming(this.forecast?.lowest, this.everyMinute);
		},
		suggestionValue(): string | null {
			const action = this.suggestion?.action;
			if (!action) return null;
			return this.$t(`battery.optimizer.action.${action}`);
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
