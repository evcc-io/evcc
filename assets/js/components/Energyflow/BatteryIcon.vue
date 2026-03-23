<template>
	<svg :style="svgStyle" viewBox="0 0 48 48">
		<path d="M0-.004h48v48H0v-48z" fill="none" />
		<path
			fill="currentColor"
			d="M35 9.996h-3v-4a2 2 0 00-2-2H18a2 2 0 00-2 2v4h-3a2 2 0 00-2 2v30a2 2 0 002 2h22a2 2 0 002-2v-30a2 2 0 00-2-2zm-15-2h8v2h-8v-2zm13 32H15v-26h18v26z"
		/>
		<path
			v-if="gridCharge"
			fill="currentColor"
			d="M17,28.652l0,-5.656l6.999,-7l7,7.001l0,5.656l-7,-7l-6.999,7Zm0,4.343l6.999,-7.001l7,7.001l0,5l-0.657,0l-6.344,-6.343l-6.343,6.343l-0.656,0l0,-5Z"
		/>
		<path
			v-else-if="hold"
			fill="currentColor"
			d="M21.943,22.053l0,9.943l-3,0l0,-9.943l3,-0Zm7,-0l0,9.943l-3,-0l0,-9.943l3,-0Z"
		/>
		<path v-else fill="currentColor" :d="socRect" />
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "BatteryIcon",
	mixins: [icon],
	props: {
		soc: { type: Number, default: 0 },
		hold: { type: Boolean, default: false },
		gridCharge: { type: Boolean, default: false },
	},
	computed: {
		socRect() {
			const height = (this.soc / (100 / 22)).toFixed(2);
			return `M30 38H18v-${height}h12v${height}z`;
		},
	},
});
</script>
