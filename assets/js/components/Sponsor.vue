<template>
	<div
		ref="sponsor"
		v-if="sponsor"
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
import confetti from "../mixins/confetti";

export default {
	name: "sponsor",
	props: {
		sponsor: String,
	},
	mixins: [confetti],
	methods: {
		surprise: function () {
			this.confetti(this.$refs.sponsor, "upLeft");
		},
	},
};
</script>

<style scoped>
.icon {
	color: var(--evcc-dark-green);
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
