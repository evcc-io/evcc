<template>
	<div v-if="activeSlot" class="text-end">
		<span class="text-nowrap">{{ day }} {{ start }}</span
		>{{ " " }}<span class="text-nowrap">– {{ end }}</span>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import type { PropType } from "vue";
import { type PriceSlot } from "../utils/forecast";
import formatter from "../mixins/formatter";

export default defineComponent({
	name: "ForecastActiveSlot",
	mixins: [formatter],
	props: {
		activeSlot: { type: Object as PropType<PriceSlot | null> },
	},
	computed: {
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
	},
});
</script>
