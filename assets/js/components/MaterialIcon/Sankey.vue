<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<!-- Tabler Icons chart-sankey, MIT, https://github.com/tabler/tabler-icons -->
		<g
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		>
			<path d="M4 6c6.944 0 9.056 8 16 8" />
			<path d="M4 12c6.37 0 9.63 6 16 6" />
			<path d="M20 6c-7.526 0 -7.905 12 -16 12" />
		</g>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "SankeyIcon",
	mixins: [icon],
});
</script>
