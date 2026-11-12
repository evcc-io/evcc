<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<!-- Tabler Icons plug, MIT, https://github.com/tabler/tabler-icons -->
		<g
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		>
			<path
				d="M9.785 6l8.215 8.215l-2.054 2.054a5.81 5.81 0 1 1 -8.215 -8.215l2.054 -2.054z"
			/>
			<path d="M4 20l3.5 -3.5" />
			<path d="M15 4l-3.5 3.5" />
			<path d="M20 9l-3.5 3.5" />
		</g>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "PlugIcon",
	mixins: [icon],
});
</script>
