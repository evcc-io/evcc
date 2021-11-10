<template>
	<div v-if="sponsor" class="my-4">
		<p>
			<fa-icon :icon="['fas', 'heart']" class="icon me-1 solid text-evcc"></fa-icon>
			{{ $t("footer.sponsor.thanks", { sponsor }) }}
		</p>
		<div class="d-flex justify-content-center">
			<button
				class="btn btn-primary mb-2 confetti-button bg-evcc"
				@click="surprise"
				ref="confetti"
			>
				{{ $t("footer.sponsor.confetti") }}
			</button>
		</div>
		<a v-if="false" href="https://evcc.io/sticker" target="_blank" class="small text-muted">{{
			$t("footer.sponsor.sticker")
		}}</a>
	</div>
	<div v-else class="my-4">
		<p>
			{{ $t("footer.sponsor.supportUs") }}
		</p>
		<div class="d-flex justify-content-center my-4">
			<a
				target="_blank"
				href="https://github.com/sponsors/andig"
				class="btn btn-outline-primary"
			>
				<fa-icon :icon="['far', 'heart']" class="icon me-1"></fa-icon>
				{{ $t("footer.sponsor.becomeSponsor") }}
			</a>
		</div>
	</div>
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
			const $el = this.$refs.confetti;
			const angle = 45 + Math.random() * 90;
			const drift = 0;

			const { top, height, left, width } = $el.getBoundingClientRect();
			const x = (left + width / 2) / window.innerWidth;
			const y = (top + height / 2) / window.innerHeight;
			const origin = { x, y };

			confetti({
				origin,
				angle,
				particleCount: 75 + Math.random() * 50,
				spread: 50 + Math.random() * 50,
				drift,
				scalar: 1.3,
				zIndex: 1056, // Bootstrap Modal is 1055
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
.confetti-button {
	/* prevent double-tap zoom */
	touch-action: none;
	user-select: none;
}
</style>
