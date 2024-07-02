<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<g
			fill="none"
			stroke="currentColor"
			stroke-linecap="round"
			stroke-linejoin="round"
			stroke-width="2"
		>
			<circle cx="8" cy="8" r="6" />
			<path d="M18.09 10.37A6 6 0 1 1 10.34 18M7 6h1v4" />
			<path d="m16.71 13.88l.7.71l-2.82 2.82" />
		</g>
	</svg>
</template>

<script>
import icon from "../../mixins/icon";

export default {
	name: "DynamicPrice",
	mixins: [icon],
};
</script>
