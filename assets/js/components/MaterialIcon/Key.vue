<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			d="M10.5 7q0-.825.588-1.412T12.5 5t1.413.588T14.5 7t-.587 1.413T12.5 9t-1.412-.587T10.5 7m2 17L8 19.5l1.5-2l-1.5-2l1.5-2.125V12.2q-1.35-.8-2.175-2.162T6.5 7q0-2.5 1.75-4.25T12.5 1t4.25 1.75T18.5 7q0 1.675-.825 3.038T15.5 12.2V21zm-4-17q0 1.4.85 2.463t2.15 1.412V14l-1.025 1.45L12 17.5l-1.375 1.775L12.5 21.15l1-1v-9.275q1.3-.35 2.15-1.412T16.5 7q0-1.65-1.175-2.825T12.5 3T9.675 4.175T8.5 7"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "Key",
	mixins: [icon],
});
</script>
