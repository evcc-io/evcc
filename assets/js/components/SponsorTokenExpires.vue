<template>
	<div v-if="expiresSoon" class="alert alert-warning my-4" role="alert">
		<i18n-t tag="div" keypath="settings.sponsorToken.expires">
			<template #inXDays>
				{{ inXDays }}
			</template>
			<template #getNewToken>
				<a href="https://sponsor.evcc.io" target="_blank" class="text-danger">
					{{ $t("settings.sponsorToken.getNew") }}
				</a>
			</template>
		</i18n-t>

		<em class="d-block mt-2" v-if="!isTrial">
			{{ $t("settings.sponsorToken.hint") }}
		</em>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";
import { TRIAL } from "./Sponsor.vue";

export default {
	name: "SponsorTokenExpires",
	mixins: [formatter],
	props: {
		expiresSoon: Boolean,
		expiresAt: String,
		name: String,
	},
	computed: {
		inXDays() {
			return this.fmtTimeAgo(new Date(this.expiresAt) - new Date());
		},
		isTrial() {
			return this.name === TRIAL;
		},
	},
};
</script>
