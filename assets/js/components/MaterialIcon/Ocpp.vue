<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			d="M3 16v-3H2q-.425 0-.712-.288T1 12t.288-.712T2 11h1V8q0-1.25.875-2.125T6 5h1q0-.425.288-.712T8 4t.713.288T9 5v14q0 .425-.288.713T8 20t-.712-.288T7 19H6q-1.25 0-2.125-.875T3 16m3 1h1V7H6q-.425 0-.712.288T5 8v8q0 .425.288.713T6 17m15-1q0 1.25-.875 2.125T18 19h-1q0 .425-.288.713T16 20t-.712-.288T15 19v-3h-3q-.425 0-.712-.288T11 15t.288-.712T12 14h3v-4h-3q-.425 0-.712-.288T11 9t.288-.712T12 8h3V5q0-.425.288-.712T16 4t.713.288T17 5h1q1.25 0 2.125.875T21 8v3h1q.425 0 .713.288T23 12t-.288.713T22 13h-1zm-4 1h1q.425 0 .713-.288T19 16V8q0-.425-.288-.712T18 7h-1zm0-5"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "Ocpp",
	mixins: [icon],
});
</script>
