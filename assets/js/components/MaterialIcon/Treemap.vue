<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<!-- Tabler Icons chart-treemap, MIT, https://github.com/tabler/tabler-icons -->
		<g
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		>
			<path
				d="M4 6a2 2 0 0 1 2 -2h12a2 2 0 0 1 2 2v12a2 2 0 0 1 -2 2h-12a2 2 0 0 1 -2 -2l0 -12"
			/>
			<path d="M12 4v16" />
			<path d="M4 15h8" />
			<path d="M12 12h8" />
			<path d="M16 12v8" />
			<path d="M16 16h4" />
		</g>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "TreemapIcon",
	mixins: [icon],
});
</script>
