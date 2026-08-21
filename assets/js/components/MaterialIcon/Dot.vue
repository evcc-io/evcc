<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<circle cx="12" cy="12" r="4" fill="currentColor" />
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "Dot",
	mixins: [icon],
});
</script>
