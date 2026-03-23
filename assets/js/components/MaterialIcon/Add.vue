<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path fill="currentColor" d="M11 13H5v-2h6V5h2v6h6v2h-6v6h-2z" />
	</svg>
</template>

<script lang="ts">
import icon from "@/mixins/icon";
import { defineComponent } from "vue";

export default defineComponent({
	name: "Add",
	mixins: [icon],
});
</script>
