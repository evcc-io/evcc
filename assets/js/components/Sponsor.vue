<template>
	<div v-if="isIndividual || isVictronDevice">
		<p class="fw-bold mb-1 d-flex">
			<shopicon-regular-heart
				class="title-icon text-primary d-inline-block me-1"
			></shopicon-regular-heart>
			{{ $t(`footer.sponsor.${isVictronDevice ? "titleVictron" : "titleSponsor"}`) }}
		</p>
		<p class="mb-3">
			{{ $t(`footer.sponsor.${isVictronDevice ? "victron" : "thanks"}`, { sponsor: name }) }}
		</p>
		<div
			class="d-flex justify-content-center align-items-center flex-column flex-lg-row align-items-lg-baseline justify-content-lg-start"
		>
			<button
				ref="confetti"
				type="button"
				class="btn btn btn-outline-primary mb-2 confetti-button bg-evcc rounded flex-shrink-0"
				@click="surprise"
			>
				<shopicon-regular-stars class="me-1 d-inline-block"></shopicon-regular-stars>
				{{ $t("footer.sponsor.confetti") }}
			</button>
			<a
				v-if="isIndividual"
				href="https://evcc.io/sticker"
				target="_blank"
				class="small text-muted ms-lg-3"
			>
				{{ $t("footer.sponsor.sticker") }}
			</a>
			<a v-else :href="sponsorLink" target="_blank" class="small text-muted ms-lg-3">
				{{ $t("footer.sponsor.becomeSponsorExtended") }}
			</a>
		</div>
	</div>
	<div v-if="!name || isTrial">
		<p class="fw-bold mb-1">
			<shopicon-regular-clock
				v-if="isTrial"
				class="title-icon text-primary d-inline-block me-1"
			></shopicon-regular-clock>
			{{ $t(`footer.sponsor.${isTrial ? "titleTrial" : "titleNoSponsor"}`) }}
		</p>
		<p class="mb-3">{{ $t(`footer.sponsor.${isTrial ? "trial" : "supportUs"}`) }}</p>
		<div
			class="d-flex justify-content-center align-items-center flex-column flex-lg-row align-items-lg-baseline justify-content-lg-start"
		>
			<a
				target="_blank"
				:href="sponsorLink"
				class="btn btn-outline-primary mb-3 become-sponsor"
			>
				<shopicon-regular-heart class="me-1 d-inline-block"></shopicon-regular-heart>
				{{ $t("footer.sponsor.becomeSponsor") }}
			</a>
			<div class="small text-muted text-center ms-lg-3">
				{{ $t("footer.sponsor.confettiPromise") }} ;)
			</div>
		</div>
	</div>
</template>

<script>
import confetti from "canvas-confetti";
import "@h2d2/shopicons/es/regular/heart";
import "@h2d2/shopicons/es/regular/stars";
import "@h2d2/shopicons/es/regular/clock";
import { docsPrefix } from "../i18n";

export const TRIAL = "trial";
export const VICTRON_DEVICE = "victron";

export default {
	name: "Sponsor",
	props: {
		name: String,
		expiresAt: String,
		expiresSoon: Boolean,
	},
	computed: {
		isTrial() {
			return this.name === TRIAL;
		},
		isVictronDevice() {
			return this.name === VICTRON_DEVICE;
		},
		isIndividual() {
			return this.name && !this.isTrial && !this.isVictronDevice;
		},
		sponsorLink() {
			return `${docsPrefix()}/docs/sponsorship`;
		},
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
.title-icon {
	transform: translateY(-2px);
}
.confetti-button {
	/* prevent double-tap zoom */
	touch-action: none;
	user-select: none;
}
.confetti-button,
.become-sponsor {
	width: 100%;
}

/* breakpoint sm */
@media (min-width: 576px) {
	.confetti-button,
	.become-sponsor {
		width: 75%;
	}
}

/* breakpoint lg */
@media (min-width: 992px) {
	.confetti-button,
	.become-sponsor {
		width: 40%;
	}
}
</style>
