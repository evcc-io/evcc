<template>
	<div :class="{ 'power-import': this.import, 'power-export': this.export }" class="d-flex">
		<shopicon-regular-powersupply class="grid flex-shrink-0"></shopicon-regular-powersupply
		><shopicon-regular-angledoublerightsmall
			class="arrow flex-shrink-0"
		></shopicon-regular-angledoublerightsmall>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/angledoublerightsmall";
import "@h2d2/shopicons/es/regular/powersupply";

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
