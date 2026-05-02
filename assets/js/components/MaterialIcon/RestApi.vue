<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			d="M5 15H3v4c0 1.1.9 2 2 2h4v-2H5v-4zM5 7h4V5H5c-1.1 0-2 .9-2 2v4h2V7zm14-2h-4v2h4v4h2V7c0-1.1-.9-2-2-2zm0 16h-4v2h4c1.1 0 2-.9 2-2v-4h-2v4zM12 9c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "RestApi",
	mixins: [icon],
});
</script>
