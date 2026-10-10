<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			transform="translate(12 3.8) scale(0.8) rotate(180) translate(-12 -9.7)"
			d="M11.288 13.712Q11 13.425 11 13V8.825L9.1 10.7q-.275.275-.687.288T7.7 10.7q-.275-.275-.275-.7t.275-.7l3.6-3.6q.15-.15.325-.212T12 5.425t.375.063t.325.212l3.6 3.6q.275.275.288.688T16.3 10.7q-.275.275-.7.275t-.7-.275L13 8.825V13q0 .425-.287.713T12 14t-.712-.288"
		/>
		<path
			fill="currentColor"
			transform="translate(12 16.2) scale(0.42) translate(-24 -24)"
			d="m43.109 14.336-12-8a2 2 0 0 0-.151-.08 2 2 0 0 0-.19-.1 2 2 0 0 0-.255-.079c-.04-.011-.077-.027-.118-.035A2 2 0 0 0 29.981 6h-12a2 2 0 0 0-.378.039c-.05.01-.1.029-.145.042a1.4 1.4 0 0 0-.49.211c-.027.016-.055.025-.081.043h-.008L4.891 14.336A2 2 0 0 0 6 18h10v5.528L8.211 39.106a2 2 0 0 0 2.649 2.7L24 35.549l13.14 6.258a2 2 0 0 0 2.649-2.7L32 23.528V18h10a2 2 0 0 0 1.109-3.664M20 10h8v4h-8Zm-7.394 4L16 11.738V14Zm16.158 12 1.151 2.3L24 31.119 18.085 28.3l1.151-2.3Zm-14.375 9.7 1.911-3.819 3.052 1.453Zm14.263-2.362 3.048-1.457 1.911 3.819ZM28 22h-8v-4h8Zm4-8v-2.263L35.4 14Z"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "BatteryDischarge",
	mixins: [icon],
});
</script>
