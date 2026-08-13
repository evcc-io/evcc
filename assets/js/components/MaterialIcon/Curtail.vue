<template>
	<svg :style="svgStyle" viewBox="0 0 48 48">
		<!-- the sun is the shopicon pv icon, cut open where the slash crosses it -->
		<mask id="curtailSlash">
			<rect width="48" height="48" fill="white" />
			<path d="M5.172 37.172 10.828 42.828 42.828 10.828 37.172 5.172Z" fill="black" />
		</mask>
		<g fill="currentColor" mask="url(#curtailSlash)">
			<path
				d="M24,34c5.514,0,10-4.486,10-10s-4.486-10-10-10c-5.514,0-10,4.486-10,10S18.486,34,24,34z M30,24c0,3.309-2.691,6-6,6s-6-2.691-6-6s2.691-6,6-6S30,20.691,30,24z"
			/>
			<rect x="22" y="4" width="4" height="6" />
			<rect x="22" y="38" width="4" height="6" />
			<rect x="4" y="22" width="6" height="4" />
			<rect x="38" y="22" width="6" height="4" />
			<rect
				x="13.5"
				y="6.278"
				transform="matrix(0.866 -0.5 0.5 0.866 -2.5623 8.9935)"
				width="4"
				height="6"
			/>
			<rect
				x="30.5"
				y="35.722"
				transform="matrix(0.866 -0.5 0.5 0.866 -15.0067 21.4372)"
				width="4.001"
				height="6"
			/>
			<rect
				x="6.277"
				y="30.5"
				transform="matrix(0.866 -0.5 0.5 0.866 -15.0068 8.9926)"
				width="6"
				height="4.001"
			/>
			<rect
				x="35.722"
				y="13.5"
				transform="matrix(0.8661 -0.4999 0.4999 0.8661 -2.5628 21.4339)"
				width="6"
				height="4.001"
			/>
			<rect
				x="7.277"
				y="12.5"
				transform="matrix(0.4999 -0.8661 0.8661 0.4999 -8.7847 15.7861)"
				width="4.001"
				height="6"
			/>
			<rect
				x="36.722"
				y="29.5"
				transform="matrix(0.5 -0.866 0.866 0.5 -8.7845 49.7849)"
				width="4.001"
				height="6"
			/>
			<rect
				x="12.5"
				y="36.722"
				transform="matrix(0.5 -0.866 0.866 0.5 -25.7843 32.7831)"
				width="6"
				height="4"
			/>
			<rect
				x="29.5"
				y="7.277"
				transform="matrix(0.5001 -0.866 0.866 0.5001 8.2139 32.7828)"
				width="5.999"
				height="4.001"
			/>
		</g>
		<path fill="currentColor" d="M6.586 38.586 9.414 41.414 41.414 9.414 38.586 6.586Z" />
	</svg>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import icon from "@/mixins/icon";

export default defineComponent({
	name: "Curtail",
	mixins: [icon],
});
</script>
