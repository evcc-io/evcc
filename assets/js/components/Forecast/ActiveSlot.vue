<template>
	<div v-if="isSlot" class="text-end">
		<span class="text-nowrap">{{ day }} {{ start }}</span
		>{{ " " }}<span class="text-nowrap">– {{ end }}</span>
	</div>
	<div v-if="isTimeseries" class="text-end">
		<span class="text-nowrap">{{ time }}</span>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import type { PropType } from "vue";
import { isPriceSlot, type PriceSlot, type TimeseriesEntry } from "../../utils/forecast.ts";
import formatter from "../../mixins/formatter.ts";

export default defineComponent({
	name: "ForecastActiveSlot",
	mixins: [formatter],
	props: {
		activeSlot: { type: Object as PropType<PriceSlot | TimeseriesEntry | null> },
	},
	computed: {
		isSlot() {
			return isPriceSlot(this.activeSlot);
		},
		isTimeseries() {
			return !isPriceSlot(this.activeSlot);
		},
		day() {
			// @ts-ignore - type checked in template
			const startDate = new Date(this.activeSlot!.start);
			return this.weekdayShort(startDate);
		},
		start() {
			// @ts-ignore - type checked in template
			const startDate = new Date(this.activeSlot!.start);
			return this.hourShort(startDate);
		},
		end() {
			// @ts-ignore - type checked in template
			const endDate = new Date(this.activeSlot!.end);
			return this.hourShort(endDate);
		},
		time() {
			// @ts-ignore - type checked in template
			const time = new Date(this.activeSlot!.ts);
			return `${this.weekdayShort(time)} ${this.hourShort(time)}`;
		},
	},
});
</script>
