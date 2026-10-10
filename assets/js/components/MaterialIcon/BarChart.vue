<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			d="M6 21q-.825 0-1.412-.587T4 19v-8q0-.825.588-1.412T6 9t1.413.588T8 11v8q0 .825-.587 1.413T6 21m6 0q-.825 0-1.412-.587T10 19V5q0-.825.588-1.412T12 3t1.413.588T14 5v14q0 .825-.587 1.413T12 21m6 0q-.825 0-1.412-.587T16 19v-4q0-.825.588-1.412T18 13t1.413.588T20 15v4q0 .825-.587 1.413T18 21"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "BarChartIcon",
	mixins: [icon],
});
</script>
