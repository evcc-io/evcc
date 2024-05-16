<template>
	<svg :style="svgStyle" viewBox="0 0 48 48">
		<path d="M0-.004h48v48H0v-48z" fill="none" />
		<path
			fill="currentColor"
			d="M35 9.996h-3v-4a2 2 0 00-2-2H18a2 2 0 00-2 2v4h-3a2 2 0 00-2 2v30a2 2 0 002 2h22a2 2 0 002-2v-30a2 2 0 00-2-2zm-15-2h8v2h-8v-2zm13 32H15v-26h18v26z"
		/>
		<path fill="currentColor" :d="socRect" />
	</svg>
</template>

<script>
import icon from "../../mixins/icon";

export default {
	name: "BatteryIcon",
	props: {
		soc: { type: Number, default: 0 },
	},
	computed: {
		socRect() {
			const height = (this.soc / (100 / 22)).toFixed(2);
			return `M30 38H18v-${height}h12v${height}z`;
		},
	},
	mixins: [icon],
};
</script>
