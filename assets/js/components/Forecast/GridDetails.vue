<template>
	<div v-if="hasBothTariffs" class="row gx-2 mt-1">
		<div class="col-6">
			<small>
				<span class="text-gray">{{ $t("main.energyflow.gridImport") }}</span>
				<br />
				<div class="d-flex flex-column flex-md-row column-gap-3 text-price fw-bold">
					<span class="text-nowrap">{{ gridSummary!.avg }}</span>
					<span v-if="gridSummary!.range" class="text-nowrap">{{
						gridSummary!.range
					}}</span>
				</div>
			</small>
		</div>
		<div class="col-6 text-end">
			<small>
				<span class="text-gray">{{ $t("main.energyflow.pvExport") }}</span>
				<div
					class="d-flex flex-column flex-md-row column-gap-3 justify-content-end text-export fw-bold"
				>
					<span class="text-nowrap">{{ feedinSummary!.avg }}</span>
					<span v-if="feedinSummary!.range" class="text-nowrap">{{
						feedinSummary!.range
					}}</span>
				</div>
				<a href="#" class="text-gray" @click.prevent="toggleFeedin">{{
					showFeedin ? $t("forecast.hideLine") : $t("forecast.showLine")
				}}</a>
			</small>
		</div>
	</div>
	<div v-else-if="gridAverage" class="row gx-2 mt-1">
		<div class="col-6">
			<small>
				<span class="text-gray">{{ $t("forecast.price.range") }}</span>
				<br />
				<span class="text-price fw-bold">{{ gridRange || gridAverage }}</span>
			</small>
		</div>
		<div class="col-6 text-end">
			<small>
				<span class="text-gray">{{ $t("forecast.price.average") }}</span>
				<br />
				<span class="text-price fw-bold">{{ gridAverage }}</span>
			</small>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import type { CURRENCY } from "@/types/evcc";
import type { ForecastSlot } from "./types";

const MAX_HOURS = 96;
const SLOTS_PER_HOUR = 4;

export default defineComponent({
	name: "GridDetails",
	mixins: [formatter],
	props: {
		grid: { type: Array as PropType<ForecastSlot[]> },
		feedin: { type: Array as PropType<ForecastSlot[]> },
		currency: { type: String as PropType<CURRENCY> },
		showFeedin: { type: Boolean, default: true },
	},
	emits: ["toggle-feedin"],
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
		gridAverage(): string {
			const upcoming = this.upcomingSlots(this.grid);
			if (upcoming.length === 0) return "";
			const avg = upcoming.reduce((a, s) => a + s.value, 0) / upcoming.length;
			return this.fmtPricePerKWh(avg, this.currency, false, true);
		},
		gridRange(): string {
			const upcoming = this.upcomingSlots(this.grid);
			if (upcoming.length === 0) return "";
			const values = upcoming.map((s) => s.value);
			const min = Math.min(...values);
			const max = Math.max(...values);
			if (min === max) return "";
			return `${this.fmtPricePerKWh(min, this.currency, false, false)} – ${this.fmtPricePerKWh(max, this.currency, false, true)}`;
		},
	},
	methods: {
		toggleFeedin() {
			this.$emit("toggle-feedin");
		},
		summarize(slots?: ForecastSlot[]): { avg: string; range: string } | null {
			const upcoming = this.upcomingSlots(slots);
			if (upcoming.length === 0) return null;
			const values = upcoming.map((s) => s.value);
			const min = Math.min(...values);
			const max = Math.max(...values);
			const avg = values.reduce((a, b) => a + b, 0) / values.length;
			const fmtAvg = this.fmtPricePerKWh(avg, this.currency, false, true);
			if (min === max) return { avg: fmtAvg, range: "" };
			const fmtMin = this.fmtPricePerKWh(min, this.currency, false, false);
			const fmtMax = this.fmtPricePerKWh(max, this.currency, false, true);
			return { avg: `⌀ ${fmtAvg}`, range: `${fmtMin} – ${fmtMax}` };
		},
		upcomingSlots(slots?: ForecastSlot[]): ForecastSlot[] {
			if (!Array.isArray(slots)) return [];
			const now = new Date();
			return slots
				.filter((slot) => new Date(slot.end) > now)
				.slice(0, MAX_HOURS * SLOTS_PER_HOUR);
		},
	},
});
</script>
