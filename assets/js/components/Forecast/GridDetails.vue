<template>
	<div v-if="hasBothTariffs" class="row gx-2 mt-1">
		<div class="col-6">
			<small>
				<button
					type="button"
					class="legend-toggle text-gray"
					:aria-pressed="showGrid"
					:disabled="!showFeedin"
					:title="showGrid ? $t('forecast.hideLine') : $t('forecast.showLine')"
					@click="$emit('toggle-line', 'grid')"
				>
					<span class="legend-dot bg-price" :class="{ inactive: !showGrid }"></span>
					<span :class="{ 'text-decoration-line-through': !showGrid }">{{
						$t("main.energyflow.gridImport")
					}}</span>
				</button>
				<br />
				<div
					class="d-flex flex-column flex-md-row column-gap-3 text-price fw-bold"
					:class="{ 'opacity-50': !showGrid }"
				>
					<span class="text-nowrap">{{ gridSummary!.avg }}</span>
					<span v-if="gridSummary!.range" class="text-nowrap">{{
						gridSummary!.range
					}}</span>
				</div>
			</small>
		</div>
		<div class="col-6 text-end">
			<small>
				<button
					type="button"
					class="legend-toggle text-gray"
					:aria-pressed="showFeedin"
					:disabled="!showGrid"
					:title="showFeedin ? $t('forecast.hideLine') : $t('forecast.showLine')"
					@click="$emit('toggle-line', 'feedin')"
				>
					<span class="legend-dot bg-export" :class="{ inactive: !showFeedin }"></span>
					<span :class="{ 'text-decoration-line-through': !showFeedin }">{{
						$t("main.energyflow.pvExport")
					}}</span>
				</button>
				<div
					class="d-flex flex-column flex-md-row column-gap-3 justify-content-end text-export fw-bold"
					:class="{ 'opacity-50': !showFeedin }"
				>
					<span class="text-nowrap">{{ feedinSummary!.avg }}</span>
					<span v-if="feedinSummary!.range" class="text-nowrap">{{
						feedinSummary!.range
					}}</span>
				</div>
			</small>
		</div>
	</div>
	<div v-else-if="gridSummary" class="row gx-2 mt-1">
		<div class="col-6">
			<small>
				<span class="text-gray">{{ $t("forecast.price.range") }}</span>
				<br />
				<span class="text-price fw-bold">{{ gridSummary.range || gridSummary.avg }}</span>
			</small>
		</div>
		<div class="col-6 text-end">
			<small>
				<span class="text-gray">{{ $t("forecast.price.average") }}</span>
				<br />
				<span class="text-price fw-bold">{{ gridSummary.avg }}</span>
			</small>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import type { CURRENCY, UiForecastSlot } from "@/types/evcc";
import { isStaticTariff } from "@/utils/forecast";

const MAX_HOURS = 96;
const SLOTS_PER_HOUR = 4;

export default defineComponent({
	name: "GridDetails",
	mixins: [formatter],
	props: {
		grid: { type: Array as PropType<UiForecastSlot[]> },
		feedin: { type: Array as PropType<UiForecastSlot[]> },
		currency: { type: String as PropType<CURRENCY> },
		showGrid: { type: Boolean, default: true },
		showFeedin: { type: Boolean, default: true },
	},
	emits: ["toggle-line"],
	computed: {
		gridSummary(): { avg: string; range: string } | null {
			return this.summarize(this.grid);
		},
		feedinSummary(): { avg: string; range: string } | null {
			return this.summarize(this.feedin);
		},
		hasBothTariffs(): boolean {
			return !!this.gridSummary && !!this.feedinSummary;
		},
	},
	methods: {
		summarize(slots?: UiForecastSlot[]): { avg: string; range: string } | null {
			const upcoming = this.upcomingSlots(slots);
			if (upcoming.length === 0) return null;
			const values = upcoming.map((s) => s.value);
			const avg = values.reduce((a, b) => a + b, 0) / values.length;
			const fmtAvg = this.fmtPricePerKWh(avg, this.currency, false, true);
			if (isStaticTariff(upcoming)) return { avg: fmtAvg, range: "" };
			const range = this.fmtPriceRange(
				Math.min(...values),
				Math.max(...values),
				this.currency
			);
			return { avg: `⌀ ${fmtAvg}`, range };
		},
		upcomingSlots(slots?: UiForecastSlot[]): UiForecastSlot[] {
			if (!Array.isArray(slots)) return [];
			const now = new Date();
			return slots
				.filter((slot) => new Date(slot.end) > now)
				.slice(0, MAX_HOURS * SLOTS_PER_HOUR);
		},
	},
});
</script>

<style scoped>
.legend-dot {
	display: inline-block;
	width: 0.6em;
	height: 0.6em;
	margin-right: 0.35em;
	border-radius: 50%;
}
.legend-dot.inactive {
	opacity: 0.25;
}
.bg-price {
	background-color: var(--evcc-price);
}
.bg-export {
	background-color: var(--evcc-export);
}
.legend-toggle {
	background: none;
	border: 0;
	padding: 0;
	font: inherit;
	cursor: pointer;
}
.legend-toggle:hover:not(:disabled) {
	text-decoration: underline;
}
.legend-toggle:disabled {
	cursor: default;
}
</style>
