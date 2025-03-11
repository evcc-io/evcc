<template>
	<div v-if="expiresSoon" class="alert alert-warning my-4" role="alert">
		<i18n-t tag="div" keypath="settings.sponsorToken.expires" scope="global">
			<template #inXDays>
				{{ inXDays }}
			</template>
			<template #getNewToken>
				<a href="https://sponsor.evcc.io" target="_blank" class="text-danger">
					{{ $t("settings.sponsorToken.getNew") }}
				</a>
			</template>
		</i18n-t>

		<em v-if="!isTrial" class="d-block mt-2">
			{{ $t("settings.sponsorToken.hint") }}
		</em>
	</div>
</template>

<script>
import formatter from "../../mixins/formatter.js";
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
