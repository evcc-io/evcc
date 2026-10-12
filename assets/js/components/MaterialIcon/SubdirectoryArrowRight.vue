<template>
	<svg :style="svgStyle" viewBox="0 2 24 24">
		<path
			fill="currentColor"
			d="M14.3 19.3q-.3-.3-.3-.712t.3-.713L17.15 15H6q-.425 0-.712-.288T5 14V5q0-.425.288-.712T6 4t.713.288T7 5v8h10.15l-2.85-2.875Q14 9.825 14 9.413t.3-.713q.3-.3.7-.3t.7.3l4.6 4.6q.15.15.212.325t.063.375t-.063.375t-.212.325l-4.6 4.6q-.3.3-.7.3t-.7-.3"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "SubdirectoryArrowRight",
	mixins: [icon],
});
</script>
