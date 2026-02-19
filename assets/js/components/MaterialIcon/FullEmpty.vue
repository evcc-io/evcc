<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<!-- horizontal line -->
		<path
			fill="currentColor"
			d="M8 20q-.425 0-.712-.288T7 19t.288-.712T8 18h8q.425 0 .713.288T17 19t-.288.713T16 20z"
			:transform="state === 'full' ? 'scale(1,-1) translate(0,-24)' : undefined"
		/>
		<!-- arrow with stem -->
		<path
			fill="currentColor"
			d="M12 16q-.425 0-.712-.288T11 15V7.825L9.1 9.7q-.275.275-.687.288T7.7 9.7q-.275-.275-.275-.7t.275-.7l3.6-3.6q.15-.15.325-.212T12 4.425t.375.063t.325.212l3.6 3.6q.275.275.288.688T16.3 9.7q-.275.275-.7.275t-.7-.275L13 7.825V15q0 .425-.287.713T12 16"
			:transform="state === 'empty' ? 'scale(1,-1) translate(0,-24)' : undefined"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "FullEmpty",
	mixins: [icon],
	props: {
		state: {
			type: String as PropType<"full" | "empty">,
			required: true,
		},
	},
});
</script>
