<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<g
			fill="none"
			stroke="currentColor"
			stroke-linecap="round"
			stroke-linejoin="round"
			stroke-width="2"
		>
			<circle cx="12" cy="12" r="9" />
			<path d="M12 8v8M8 12h8" />
		</g>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "AddCircle",
	mixins: [icon],
});
</script>
