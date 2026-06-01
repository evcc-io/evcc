<template>
	<div v-if="average" class="row gx-2 mt-1">
		<div class="col-6">
			<small>
				<span class="text-gray">{{ $t("forecast.co2.range") }}</span>
				<br />
				<span class="text-co2 fw-bold">{{ range }}</span>
			</small>
		</div>
		<div class="col-6 text-end">
			<small>
				<span class="text-gray">{{ $t("forecast.co2.average") }}</span>
				<br />
				<span class="text-co2 fw-bold">{{ average }}</span>
			</small>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import type { ForecastSlot } from "./types";

const MAX_HOURS = 96;
const SLOTS_PER_HOUR = 4;

export default defineComponent({
	name: "Co2Details",
	mixins: [formatter],
	props: {
		co2: { type: Array as PropType<ForecastSlot[]> },
	},
	computed: {
		upcomingSlots(): ForecastSlot[] {
			if (!Array.isArray(this.co2)) return [];
			const now = new Date();
			return this.co2
				.filter((slot) => new Date(slot.end) > now)
				.slice(0, MAX_HOURS * SLOTS_PER_HOUR);
		},
		average(): string {
			if (this.upcomingSlots.length === 0) return "";
			const avg =
				this.upcomingSlots.reduce((a, s) => a + s.value, 0) / this.upcomingSlots.length;
			return this.fmtCo2Medium(avg);
		},
		range(): string {
			if (this.upcomingSlots.length === 0) return "";
			const values = this.upcomingSlots.map((s) => s.value);
			const min = Math.min(...values);
			const max = Math.max(...values);
			return `${this.fmtNumber(min, 0)} – ${this.fmtCo2Medium(max)}`;
		},
	},
});
</script>
