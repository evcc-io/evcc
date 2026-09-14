<template>
	<svg :style="svgStyle" viewBox="0 -960 960 960">
		<path fill="currentColor" d="M504-480 320-664l56-56 240 240-240 240-56-56 184-184Z" />
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "ChevronRight",
	mixins: [icon],
});
</script>
