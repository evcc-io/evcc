<template>
	<div
		v-if="limits.length"
		class="limit-stripe small rounded-4 py-2 px-3 mb-3 d-sm-flex flex-wrap align-items-baseline justify-content-between gap-sm-2"
		:class="{ 'limit-stripe--critical': critical }"
		data-testid="hems-warning"
	>
		<div class="limit-title fw-bold text-uppercase mb-2 mb-sm-0">
			{{ $t("main.hemsWarning.title") }}
		</div>
		<div class="d-flex flex-column gap-1 flex-sm-row gap-sm-4">
			<div
				v-for="limit in limits"
				:key="limit.name"
				class="d-flex align-items-baseline gap-2"
			>
				<span class="fw-medium">{{ $t(`main.hemsWarning.${limit.name}`) }}</span>
				<span class="fw-bold tabular text-nowrap">&le; {{ limit.value }}</span>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import type { HemsStatus } from "@/types/evcc";
import formatter, { POWER_UNIT } from "@/mixins/formatter";

// share of the limit that has to be used before the limit is shown at all
const WARNING_DEGREE = 0.7;
// share of the limit that makes it critical
const CRITICAL_DEGREE = 0.9;

interface Limit {
	name: string;
	value: string;
	degree: number;
}

export default defineComponent({
	name: "HemsWarning",
	mixins: [formatter],
	props: {
		status: { type: Object as PropType<HemsStatus> },
		gridPower: { type: Number, default: 0 },
	},
	computed: {
		limits(): Limit[] {
			const status = this.status;
			if (!status) {
				return [];
			}
			const result: Limit[] = [];
			if (status.dimmed && status.maxConsumptionPower !== undefined) {
				result.push(this.limit("consumption", status.maxConsumptionPower, this.gridPower));
			}
			if ((status.curtailed ?? 100) < 100 && status.maxProductionPower !== undefined) {
				result.push(this.limit("feedIn", status.maxProductionPower, -this.gridPower));
			}
			return result.filter(({ degree }) => degree >= WARNING_DEGREE);
		},
		critical(): boolean {
			return this.limits.some(({ degree }) => degree >= CRITICAL_DEGREE);
		},
	},
	methods: {
		limit(name: string, limit: number, power: number): Limit {
			// a zero limit is fully used as soon as power flows in its direction
			const degree = limit > 0 ? power / limit : Number(power > 0);
			return { name, value: this.fmtW(limit, POWER_UNIT.KW), degree };
		},
	},
});
</script>

<style scoped>
/* animated throttle stripe, echoes the charging bar */
.limit-stripe {
	--limit-color: var(--evcc-dark-yellow);
	background-color: color-mix(in srgb, var(--limit-color) 9%, transparent);
	background-image: repeating-linear-gradient(
		-45deg,
		color-mix(in srgb, var(--limit-color) 20%, transparent) 0 8px,
		transparent 8px 20px
	);
	background-size: 28.28px 28.28px;
}

.limit-stripe--critical {
	--limit-color: var(--evcc-orange);
}

.limit-title {
	color: var(--limit-color);
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
