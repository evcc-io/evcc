<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			d="M6.5 20q-2.275 0-3.887-1.575T1 14.575q0-1.95 1.175-3.475T5.25 9.15q.625-2.3 2.5-3.725T12 4q2.4 0 4.238 1.4T18.725 9q.125.425-.112.75t-.588.425t-.725-.075t-.55-.675q-.5-1.5-1.8-2.463T12 6Q9.925 6 8.463 7.463T7 11h-.5q-1.45 0-2.475 1.025T3 14.5t1.025 2.475T6.5 18H13q.425 0 .713.288T14 19t-.288.713T13 20zM17 20q-.425 0-.712-.288T16 19v-3q0-.425.288-.712T17 15v-1q0-.825.588-1.412T19 12t1.413.588T21 14v1q.425 0 .713.288T22 16v3q0 .425-.288.713T21 20zm1-5h2v-1q0-.425-.288-.712T19 13t-.712.288T18 14z"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "RemoteAccessIcon",
	mixins: [icon],
});
</script>
