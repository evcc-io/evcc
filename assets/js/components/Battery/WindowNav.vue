<template>
	<div class="d-flex align-items-center gap-1">
		<DateNavigatorButton
			prev
			:disabled="prevDisabled"
			:on-click="() => $emit('prev')"
			data-testid="battery-chart-prev"
		/>
		<span class="text-center text-truncate evcc-default-text" style="width: 6em">{{
			label
		}}</span>
		<DateNavigatorButton
			next
			:disabled="nextDisabled"
			:on-click="() => $emit('next')"
			data-testid="battery-chart-next"
		/>
	</div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import DateNavigatorButton from "../Sessions/DateNavigatorButton.vue";

export default defineComponent({
	name: "WindowNav",
	components: { DateNavigatorButton },
	props: {
		label: { type: String, default: "" },
		prevDisabled: Boolean,
		nextDisabled: Boolean,
	},
	emits: ["prev", "next"],
});
</script>
