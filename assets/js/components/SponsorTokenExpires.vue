<template>
	<div v-if="expiresSoon" class="alert alert-warning" role="alert">
		<i18n-t tag="div" class="mb-2" keypath="settings.sponsorToken.expires">
			<template #inXDays>
				{{ inXDays }}
			</template>
			<template #getNewToken>
				<a href="https://sponsor.evcc.io" target="_blank" class="text-danger">
					{{ $t("settings.sponsorToken.getNew") }}
				</a>
			</template>
		</i18n-t>

		<em>
			{{ $t("settings.sponsorToken.hint") }}
		</em>
	</div>
</template>

<script>
import formatter from "../mixins/formatter";

export default {
	name: "SponsorTokenExpires",
	mixins: [formatter],
	props: {
		expiresSoon: Boolean,
		expiresAt: String,
	},
	computed: {
		inXDays() {
			return this.fmtTimeAgo(new Date(this.expiresAt) - new Date());
		},
	},
};
</script>
