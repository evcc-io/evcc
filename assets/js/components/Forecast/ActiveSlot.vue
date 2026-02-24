<template>
	<div v-if="isSlot" class="text-end tabular">
		<span class="text-nowrap">{{ day }} {{ start }}</span
		>{{ " " }}<span class="text-nowrap">– {{ end }}</span>
	</div>
	<div v-if="isTimeseries" class="text-end tabular">
		<span class="text-nowrap">{{ time }}</span>
	</div>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import formatter from "@/mixins/formatter";
import { isForecastSlot, type ForecastSlot, type TimeseriesEntry } from "./types";

export default defineComponent({
	name: "ForecastActiveSlot",
	mixins: [formatter],
	props: {
		activeSlot: { type: Object as PropType<ForecastSlot | TimeseriesEntry | null> },
	},
	computed: {
		isSlot() {
			return this.activeSlot !== null && isForecastSlot(this.activeSlot);
		},
		isTimeseries() {
			return this.activeSlot !== null && !isForecastSlot(this.activeSlot);
		},
		day() {
			const startDate = new Date((this.activeSlot! as ForecastSlot).start);
			return this.weekdayShort(startDate);
		},
		start() {
			const startDate = new Date((this.activeSlot! as ForecastSlot).start);
			return this.fmtHourMinute(startDate);
		},
		end() {
			const endDate = new Date((this.activeSlot! as ForecastSlot).end);
			return this.fmtHourMinute(endDate);
		},
		time() {
			const time = new Date((this.activeSlot! as TimeseriesEntry).ts);
			return `${this.weekdayShort(time)} ${this.fmtHourMinute(time)}`;
		},
	},
});
</script>
