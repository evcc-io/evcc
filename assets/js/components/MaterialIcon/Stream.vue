<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<!-- a curve and its mirror image, the symmetric outline of a stream chart -->
		<g
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		>
			<path d="M4 10.5c3 0 4 -4.5 7 -4.5s3.5 3 5.5 3s2.5 -3 3.5 -4" />
			<path d="M4 13.5c3 0 4 4.5 7 4.5s3.5 -3 5.5 -3s2.5 3 3.5 4" />
		</g>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "StreamIcon",
	mixins: [icon],
});
</script>
