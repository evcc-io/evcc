<template>
	<div
		v-if="sponsor"
		ref="sponsor"
		class="btn btn-link pe-0 text-decoration-none link-dark text-nowrap sponsor-button"
		@click.stop.prevent="surprise"
	>
		<span class="d-inline d-sm-none">{{ $t("footer.sponsor.sponsoredShort") }}</span>
		<span class="d-none d-sm-inline">{{
			$t("footer.sponsor.sponsoredLong", { sponsor })
		}}</span>
		<fa-icon :icon="['fas', 'heart']" class="icon ms-1"></fa-icon>
	</div>
	<a
		v-else
		href="https://github.com/sponsors/andig"
		target="_blank"
		class="btn btn-link pe-0 text-decoration-none link-dark text-nowrap support-button"
	>
		<span class="d-inline d-sm-none">{{ $t("footer.sponsor.supportProjectShort") }}</span>
		<span class="d-none d-sm-inline">{{ $t("footer.sponsor.supportProjectLong") }}</span>
		<fa-icon :icon="['far', 'heart']" class="icon ms-1 outline"></fa-icon>
		<fa-icon :icon="['fas', 'heart']" class="icon ms-1 solid"></fa-icon>
	</a>
</template>

<script>
import confetti from "canvas-confetti";

export default {
	name: "Sponsor",
	props: {
		sponsor: String,
	},
	methods: {
		surprise: function () {
			console.log(this.$refs.sponsor);
			const { top, height, left, width } = this.$refs.sponsor.getBoundingClientRect();
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
	display: inline-block;
}
.sponsor-button {
	/* prevent double-tap zoom */
	touch-action: none;
	user-select: none;
}
.support-button .solid {
	display: none;
}
.support-button:hover .solid {
	display: inline-block;
}
.support-button:hover .outline {
	display: none;
}
</style>
