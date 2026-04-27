<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			d="M3 16v-3H2q-.425 0-.712-.288T1 12t.288-.712T2 11h1V8q0-1.25.875-2.125T6 5h1q0-.425.288-.712T8 4t.713.288T9 5v14q0 .425-.288.713T8 20t-.712-.288T7 19H6q-1.25 0-2.125-.875T3 16m3 1h1V7H6q-.425 0-.712.288T5 8v8q0 .425.288.713T6 17m8.5-5l3.5-3.5V11h3v2h-3v2.5zm3.5 5v-2h-4v-2h4v-2l3 3z"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "OcppForwarder",
	mixins: [icon],
});
</script>
