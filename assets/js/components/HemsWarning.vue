<template>
	<div
		v-if="consumptionLimit || feedInLimit"
		class="limit-stripe small rounded-4 py-2 px-3 mb-3 d-sm-flex flex-wrap align-items-baseline justify-content-between gap-sm-2"
		data-testid="hems-warning"
	>
		<div class="fw-bold text-uppercase text-warning mb-2 mb-sm-0">
			{{ $t("main.hemsWarning.title") }}
		</div>
		<div class="d-flex flex-column gap-1 flex-sm-row gap-sm-4">
			<div v-if="consumptionLimit" class="d-flex align-items-baseline gap-2">
				<span class="fw-medium">{{ $t("main.hemsWarning.consumption") }}</span>
				<span class="fw-bold tabular text-nowrap">&le; {{ consumptionLimit }}</span>
			</div>
			<div v-if="feedInLimit" class="d-flex align-items-baseline gap-2">
				<span class="fw-medium">{{ $t("main.hemsWarning.feedIn") }}</span>
				<span class="fw-bold tabular text-nowrap">&le; {{ feedInLimit }}</span>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import type { HemsStatus } from "@/types/evcc";
import formatter, { POWER_UNIT } from "@/mixins/formatter";

export default defineComponent({
	name: "HemsWarning",
	mixins: [formatter],
	props: {
		status: { type: Object as PropType<HemsStatus> },
	},
	computed: {
		consumptionLimit(): string | null {
			return this.status?.dimmed && this.status.maxConsumptionPower
				? this.fmtW(this.status.maxConsumptionPower, POWER_UNIT.KW)
				: null;
		},
		feedInLimit(): string | null {
			return this.status?.curtailed && this.status.maxProductionPower !== undefined
				? this.fmtW(this.status.maxProductionPower, POWER_UNIT.KW)
				: null;
		},
	},
});
</script>

<style scoped>
/* animated throttle stripe, echoes the charging bar */
.limit-stripe {
	background-color: color-mix(in srgb, var(--evcc-orange) 9%, transparent);
	background-image: repeating-linear-gradient(
		-45deg,
		color-mix(in srgb, var(--evcc-orange) 20%, transparent) 0 8px,
		transparent 8px 20px
	);
	background-size: 28.28px 28.28px;
}

@media (prefers-reduced-motion: no-preference) {
	.limit-stripe {
		animation: limit-stripe-move 1.5s linear infinite;
	}
}

@keyframes limit-stripe-move {
	to {
		background-position: 28.28px 0;
	}
}
</style>
