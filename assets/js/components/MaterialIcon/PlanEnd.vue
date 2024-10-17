<template>
	<svg :style="svgStyle" viewBox="0 0 24 24">
		<path
			fill="currentColor"
			d="M21 18q-.425 0-.712-.288T20 17V7q0-.425.288-.712T21 6t.713.288T22 7v10q0 .425-.288.713T21 18m-6.825-5H3q-.425 0-.712-.288T2 12t.288-.712T3 11h11.175L11.3 8.1q-.275-.275-.288-.687T11.3 6.7q.275-.275.7-.275t.7.275l4.6 4.6q.15.15.213.325t.062.375t-.062.375t-.213.325l-4.6 4.6q-.275.275-.687.275T11.3 17.3q-.3-.3-.3-.712t.3-.713z"
		/>
	</svg>
</template>

<script>
import icon from "../../mixins/icon";

export default {
	name: "PlanEnd",
	mixins: [icon],
};
</script>
