<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			d="M19.35 10.04A7.49 7.49 0 0 0 5.5 8A5.5 5.5 0 0 0 6 19h4v-6H8l4-4l4 4h-2v6h5a5 5 0 0 0 .35-8.96"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "FtpBackupIcon",
	mixins: [icon],
});
</script>
