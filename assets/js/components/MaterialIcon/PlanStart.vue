<template>
	<svg :style="svgStyle" viewBox="0 0 48 48">
		<path
			fill="currentColor"
			d="M26.585,42.698l-2.585,-0l0,0.002c-11.046,0 -20,-8.954 -20,-20c0,-11.046 8.954,-20 20,-20c11.046,-0 20,8.954 20,20c0,6.041 -2.678,11.456 -6.912,15.124l-2.843,-2.843c3.515,-2.938 5.755,-7.353 5.755,-12.281c-0,-8.822 -7.178,-16 -16,-16c-8.822,-0 -16,7.178 -16,16c-0,8.816 7.168,15.99 15.982,16l0,-0.002l2.682,-0l-2.282,-2.282l2.828,-2.829l7.071,7.071l-7.071,7.072l-2.828,-2.829l2.203,-2.203Z"
		/>
		<path
			fill="currentColor"
			d="M26,12.7l-4,-0l0,12l12,-0l0,-4l-8,-0l0,-8Z"
			style="fill-rule: nonzero"
		/>
	</svg>
</template>

<script>
import icon from "../../mixins/icon";

export default {
	name: "PlanStart",
	mixins: [icon],
};
</script>
