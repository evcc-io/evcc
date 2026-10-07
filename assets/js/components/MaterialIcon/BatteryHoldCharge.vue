<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			transform="rotate(180 12 12)"
			d="M8 19q-.425 0-.712-.288T7 18t.288-.712T8 17h8q.425 0 .713.288T17 18t-.288.713T16 19zM11 17V8.825L9.1 10.7q-.275.275-.687.288T7.7 10.7q-.275-.275-.275-.7t.275-.7l3.6-3.6q.15-.15.325-.212T12 5.425t.375.063t.325.212l3.6 3.6q.275.275.288.688T16.3 10.7q-.275.275-.7.275t-.7-.275L13 8.825V17z"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "BatteryHoldCharge",
	mixins: [icon],
});
</script>
