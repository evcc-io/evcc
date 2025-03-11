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
import { type PriceSlot, type TimeseriesEntry, type EventEntry } from "../utils/forecast";
import formatter from "../mixins/formatter";

export default defineComponent({
	name: "ForecastActiveSlot",
	mixins: [formatter],
	props: {
		activeSlot: { type: Object as PropType<PriceSlot | TimeseriesEntry | EventEntry | null> },
	},
	computed: {
		isSlot() {
			return this.activeSlot?.start && this.activeSlot?.end;
		},
		isTimeseries() {
			return this.activeSlot?.ts;
		},
		day() {
			const startDate = new Date(this.activeSlot!.start);
			return this.weekdayShort(startDate);
		},
		start() {
			const startDate = new Date(this.activeSlot!.start);
			return this.hourShort(startDate);
		},
		end() {
			const endDate = new Date(this.activeSlot!.end);
			return this.hourShort(endDate);
		},
		time() {
			const time = new Date(this.activeSlot!.ts);
			return `${this.weekdayShort(time)} ${this.hourShort(time)}`;
		},
	},
});
</script>
