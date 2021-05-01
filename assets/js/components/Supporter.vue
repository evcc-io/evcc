<template>
	<div
		ref="supporter"
		v-if="supporter"
		class="btn btn-link pe-0 text-decoration-none link-dark text-nowrap supporter-button"
		@click.stop.prevent="surprise"
	>
		<span>Supporter</span>
		<fa-icon icon="star" class="icon ms-1"></fa-icon>
	</div>
	<a
		v-else
		href="https://github.com/sponsors/andig"
		target="_blank"
		class="btn btn-link pe-0 text-decoration-none link-dark text-nowrap"
	>
		<span class="d-none d-sm-inline">Projekt </span>unterstützen
		<fa-icon icon="heart" class="icon ms-1"></fa-icon>
	</a>
</template>

<script>
import confetti from "canvas-confetti";

export default {
	name: "Supporter",
	props: {
		supporter: Boolean,
	},
	methods: {
		surprise: function () {
			console.log(this.$refs.supporter);
			const { top, height, left, width } = this.$refs.supporter.getBoundingClientRect();
			const x = (left + width / 2) / window.innerWidth;
			const y = (top + height / 2) / window.innerHeight;
			const origin = { x, y };

			confetti({
				origin,
				angle: 90 + Math.random() * 35,
				particleCount: 75 + Math.random() * 50,
				spread: 50 + Math.random() * 50,
				drift: -0.5,
				scalar: 1.3,
				colors: [
					"#0d6efd",
					"#0fdd42",
					"#408458",
					"#4923BA",
					"#5BC8EC",
					"#C54482",
					"#CC444A",
					"#EE8437",
					"#F7C144",
					"#FFFD54",
				],
			});
		},
	},
};
</script>

<style scoped>
.icon {
	color: #0fdd42;
}
.supporter-button {
	/* prevent double-tap zoom */
	touch-action: none;
	user-select: none;
}
</style>
