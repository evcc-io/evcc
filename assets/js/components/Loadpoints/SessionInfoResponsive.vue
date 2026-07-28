<template>
	<div class="d-flex align-items-start justify-content-end flex-wrap">
		<SessionInfo
			v-for="key in fixedKeys"
			:key="key"
			v-bind="$props"
			:fixed-key="key"
			class="ms-4"
		/>
		<SessionInfo
			v-if="remainingKeys.length"
			v-bind="$props"
			:only-keys="remainingKeys"
			class="ms-4"
		/>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import SessionInfo from "./SessionInfo.vue";
import breakpoint, { type Breakpoint } from "@/mixins/breakpoint";
import { availableSessionInfoKeys } from "./sessionInfoKeys";
import type { CURRENCY, SessionInfoKey } from "@/types/evcc";

// how many stats are shown as fixed (non-dropdown) columns at each breakpoint;
// at "xxl" all available stats are always shown and the dropdown disappears
const FIXED_COUNT_BY_BREAKPOINT: Record<Breakpoint, number> = {
	xs: 0,
	sm: 0,
	md: 2,
	lg: 4,
	xl: 6,
	xxl: Infinity,
};

export default defineComponent({
	name: "SessionInfoResponsive",
	components: { SessionInfo },
	mixins: [breakpoint],
	props: {
		id: String,
		sessionCo2PerKWh: { type: Number, default: 0 },
		sessionPricePerKWh: { type: Number, default: 0 },
		sessionPrice: { type: Number, default: 0 },
		currency: String as PropType<CURRENCY>,
		sessionSolarPercentage: { type: Number, default: 0 },
		chargeRemainingDurationInterpolated: { type: Number, default: 0 },
		chargeDurationInterpolated: Number,
		sessionEnergy: { type: Number, default: 0 },
		last24hEnergy: Number,
		last7dEnergy: Number,
		tariffCo2: Number,
		tariffGrid: Number,
	},
	computed: {
		availableKeys(): SessionInfoKey[] {
			return availableSessionInfoKeys({
				chargeRemainingDurationInterpolated: this.chargeRemainingDurationInterpolated,
				tariffGrid: this.tariffGrid,
				tariffCo2: this.tariffCo2,
				last24hEnergy: this.last24hEnergy,
				last7dEnergy: this.last7dEnergy,
			});
		},
		fixedCount(): number {
			return FIXED_COUNT_BY_BREAKPOINT[this.breakpoint as Breakpoint] ?? 0;
		},
		fixedKeys(): SessionInfoKey[] {
			return this.availableKeys.slice(0, this.fixedCount);
		},
		remainingKeys(): SessionInfoKey[] {
			return this.availableKeys.slice(this.fixedCount);
		},
	},
});
</script>
