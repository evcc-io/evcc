<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			d="M5.5 17.5q-2.3 0-3.9-1.6T0 12t1.6-3.9t3.9-1.6q.925 0 1.775.325T8.8 7.75l1.7 1.55L9 10.65l-1.55-1.4q-.4-.35-.9-.55T5.5 8.5q-1.45 0-2.475 1.025T2 12t1.025 2.475T5.5 15.5q.55 0 1.05-.2t.9-.55l7.75-7q.675-.6 1.525-.925T18.5 6.5q2.3 0 3.9 1.6T24 12t-1.6 3.9t-3.9 1.6q-.925 0-1.775-.325T15.2 16.25l-1.7-1.55l1.5-1.35l1.55 1.4q.4.35.9.55t1.05.2q1.45 0 2.475-1.025T22 12t-1.025-2.475T18.5 8.5q-.55 0-1.05.2t-.9.55l-7.75 7q-.675.6-1.525.925T5.5 17.5"
		/>
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "AlwaysCharge",
	mixins: [icon],
});
</script>
