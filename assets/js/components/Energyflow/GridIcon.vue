<template>
	<div :class="{ 'power-import': this.import, 'power-export': this.export }">
		<svg
			viewBox="0 0 512 512"
			xmlns="http://www.w3.org/2000/svg"
			xml:space="preserve"
			class="grid svg-inline--fa fa-home fa-w-18"
			style="
				fill-rule: evenodd;
				clip-rule: evenodd;
				stroke-linejoin: round;
				stroke-miterlimit: 2;
			"
		>
			<path
				fill="currentColor"
				d="M166.815 224.933h-64.054a11.55 11.55 0 0 0-.916-.037h-22.5c-6.209 0-11.25 5.041-11.25 11.25v4.14a16.22 16.22 0 0 0-.011.585v31.876c0 .197.004.392.011.586v33.406c0 6.209 5.041 11.25 11.25 11.25h22.5c6.209 0 11.25-5.041 11.25-11.25v-18.053h29.592C105.897 385.94 72.74 473.78 72.74 473.78c-3.4 8.328.46 17.901 8.61 21.375l29.51 12.578c8.15 3.474 17.52-.47 20.91-8.797l49.19-120.719h150.07l49.2 120.719c3.4 8.337 12.76 12.271 20.91 8.797l29.51-12.578c8.15-3.474 12-13.047 8.61-21.375 0 0-33.103-87.922-69.773-185.094H398.2v18.003c0 6.257 5.08 11.337 11.337 11.337h22.674c6.257 0 11.337-5.08 11.337-11.337V236.27c0-6.257-5.08-11.337-11.337-11.337h-86.794c-6.992-18.51-13.88-36.732-20.464-54.135H398.2v18.003c0 6.257 5.08 11.337 11.337 11.337h22.674c6.257 0 11.337-5.08 11.337-11.337v-70.419c0-6.257-5.08-11.337-11.337-11.337H300.803c-15.742-41.475-26.669-70.049-27.848-72.584-16.553-35.597-17.28-36.651-33.105-1.138-1.386 3.108-12.466 32.042-28.319 73.722h-108.77a11.55 11.55 0 0 0-.916-.037h-22.5c-6.209 0-11.25 5.041-11.25 11.25v4.14a16.22 16.22 0 0 0-.011.585v31.877c0 .196.004.391.011.585v33.406c0 6.209 5.041 11.25 11.25 11.25h22.5c6.209 0 11.25-5.041 11.25-11.25v-18.053h74.232a139451.94 139451.94 0 0 0-20.512 54.135Zm129.566 63.753 7.999 24.139h-96.76l8.189-24.139h80.572Zm-21.126-63.753h-37.817l18.366-54.135h1.512l17.939 54.135Z"
				style="fill-rule: nonzero"
			/>
		</svg>
		<fa-icon class="arrow" icon="angle-double-right"></fa-icon>
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
	transform: translateX(0rem);
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
	transform: translateX(-1.1rem) scaleX(1);
}
.power-import .arrow {
	opacity: 1;
	transform: translateX(-0.1rem) scaleX(1);
}
</style>
