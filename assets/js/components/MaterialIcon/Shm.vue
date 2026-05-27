<template>
	<svg :style="svgStyle" viewBox="0 0 32 32">
		<path
			d="M3.046,6.43c-1.365,-0 -2.513,1.149 -2.513,2.513l0,15.66c2.847,-1.757 15.172,-8.696 30.933,-8.411l0,-9.762l-28.42,-0Z"
			fill="currentColor"
		/>
		<path
			d="M1.5,25.57l27.454,0c1.365,0 2.513,-1.149 2.513,-2.513l-0.001,-5.568c-14.933,-0.278 -26.74,6.137 -29.965,8.082"
			fill="currentColor"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "Shm",
	mixins: [icon],
});
</script>
