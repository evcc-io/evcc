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
import { isForecastSlot, type ForecastSlot, type TimeseriesEntry } from "../../utils/forecast";
import formatter from "../../mixins/formatter";

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
			return this.hourShort(startDate);
		},
		end() {
			const endDate = new Date((this.activeSlot! as ForecastSlot).end);
			return this.hourShort(endDate);
		},
		time() {
			const time = new Date((this.activeSlot! as TimeseriesEntry).ts);
			return `${this.weekdayShort(time)} ${this.hourShort(time)}`;
		},
	},
});
</script>
