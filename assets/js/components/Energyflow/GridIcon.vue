<template>
	<div :class="{ 'power-import': this.import, 'power-export': this.export }">
		<svg
			viewBox="0 0 512 512"
			xmlns="http://www.w3.org/2000/svg"
			xml:space="preserve"
			class="grid svg-inline--fa fa-home fa-w-18"
		>
			<path
				fill="currentColor"
				d="M217.529 46.629H83.817c-8.83 0-16 7.169-16 16v32c0 .196.004.391.011.586v51.22c0 6.209 5.041 11.25 11.25 11.25h22.5c6.209 0 11.25-5.041 11.25-11.25v-35.806h104.701v84.697H83.577c-8.83 0-16 7.169-16 16v32c0 .196.004.391.011.586v51.22c0 6.209 5.041 11.25 11.25 11.25h22.5c6.209 0 11.25-5.041 11.25-11.25v-35.806h104.941v49.925L176.47 506.932l160 .033-39.179-197.538v-50.101h100.402v35.756c0 6.257 5.08 11.337 11.337 11.337h22.674c6.257 0 11.337-5.08 11.337-11.337v-83.756c0-8.831-7.169-16-16-16h-129.75v-84.697h100.642v35.756c0 6.257 5.08 11.337 11.337 11.337h22.674c6.257 0 11.337-5.08 11.337-11.337V62.629c0-8.831-7.169-16-16-16h-129.99V26.162c-.043-12.777-40.058-18.846-40.159-18.633 0 0-39.66 6.092-39.603 18.633v20.467Z"
			/></svg
		><fa-icon class="arrow" icon="angle-double-right"></fa-icon>
	</div>
</template>

<script>
export default {
	name: "GridIcon",
	props: {
		import: { type: Boolean },
		export: { type: Boolean },
	},
};
</script>

<style scoped>
.grid {
	transform: translateX(0.35rem);
	transition-property: transform;
	transition-duration: 250ms;
	transition-timing-function: ease;
}
.power-export .grid {
	transform: translateX(0.9rem);
}
.power-import .grid {
	transform: translateX(-0.1rem);
}
.arrow {
	margin-left: -0.2rem;
	opacity: 0;
	transform: translateX(-0.5rem);
	transition-property: opacity, transform;
	transition-duration: 250ms;
	transition-timing-function: ease;
}
.power-export .arrow {
	opacity: 1;
	transform: translateX(-0.9rem) scaleX(1);
}
.power-import .arrow {
	opacity: 1;
	transform: translateX(0.1em) scaleX(1);
}
</style>
