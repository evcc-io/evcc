<template>
	<svg :style="svgStyle" viewBox="0 0 48 48">
		<path
			fill="currentColor"
			d="M22,43.901c-10.107,-1.003 -18,-9.53 -18,-19.901c0,-11.046 8.954,-20 20,-20c11.046,-0 20,8.954 20,20c0,5.523 -2.239,10.523 -5.858,14.142l-2.834,-2.834c2.897,-2.897 4.692,-6.897 4.692,-11.308c0,-8.822 -7.178,-16 -16,-16c-8.822,-0 -16,7.178 -16,16c0,8.145 6.118,14.888 14,15.876l0,-2.876l4,0l0,10l-4,0l-0,-3.099Z"
		/>
		<path
			fill="currentColor"
			d="M26,14l-4,0l0,12l12,0l0,-4l-8,0l0,-8Z"
			style="fill-rule: nonzero"
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
