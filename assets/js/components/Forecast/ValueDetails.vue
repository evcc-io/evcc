<template>
	<div v-if="average" class="row gx-2 mt-1">
		<div class="col-6">
			<small>
				<span class="text-gray">{{ $t(`forecast.${type}.range`) }}</span>
				<br />
				<span :class="`text-${type}`" class="fw-bold">{{ range }}</span>
			</small>
		</div>
		<div class="col-6 text-end">
			<small>
				<span class="text-gray">{{ $t(`forecast.${type}.average`) }}</span>
				<br />
				<span :class="`text-${type}`" class="fw-bold">{{ average }}</span>
			</small>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import type { ForecastSlot } from "./types";
import type { ValueChartType } from "./ValueChart.vue";

const MAX_HOURS = 96;
const SLOTS_PER_HOUR = 4;

export default defineComponent({
	name: "ValueDetails",
	mixins: [formatter],
	props: {
		type: { type: String as PropType<ValueChartType>, required: true },
		rates: { type: Array as PropType<ForecastSlot[]> },
	},
	computed: {
		upcomingSlots(): ForecastSlot[] {
			if (!Array.isArray(this.rates)) return [];
			const now = new Date();
			return this.rates
				.filter((slot) => new Date(slot.end) > now)
				.slice(0, MAX_HOURS * SLOTS_PER_HOUR);
		},
		average(): string {
			if (this.upcomingSlots.length === 0) return "";
			const avg =
				this.upcomingSlots.reduce((a, s) => a + s.value, 0) / this.upcomingSlots.length;
			return this.fmtValue(avg);
		},
		range(): string {
			if (this.upcomingSlots.length === 0) return "";
			const values = this.upcomingSlots.map((s) => s.value);
			const min = Math.min(...values);
			const max = Math.max(...values);
			if (this.type === "co2") {
				return `${this.fmtNumber(min, 0)} – ${this.fmtCo2Medium(max)}`;
			}
			return `${this.fmtValue(min)} – ${this.fmtValue(max)}`;
		},
	},
	methods: {
		fmtValue(value: number): string {
			return this.type === "co2" ? this.fmtCo2Medium(value) : this.fmtTemperature(value);
		},
	},
});
</script>
