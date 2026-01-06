<template>
	<div v-if="expiresSoon" class="alert alert-warning my-4" role="alert">
		<div>
			<i18n-t tag="span" keypath="settings.sponsorToken.expires" scope="global">
				<template #inXDays>
					{{ inXDays }}
				</template>
			</i18n-t>
			{{ " " }}
			<i18n-t
				tag="span"
				:keypath="
					fromYaml
						? 'settings.sponsorToken.expiresUpdateYaml'
						: 'settings.sponsorToken.expiresUpdateUi'
				"
				scope="global"
			>
				<template #getNewToken>
					<a href="https://sponsor.evcc.io" target="_blank" class="text-danger">
						{{ $t("settings.sponsorToken.getNewToken") }}
					</a>
				</template>
			</i18n-t>
		</div>

		<em v-if="!isTrial" class="d-block mt-2">
			{{ $t("settings.sponsorToken.hint") }}
		</em>
	</div>
</template>

<script lang="ts">
import formatter from "@/mixins/formatter";
import { TRIAL } from "./Sponsor.vue";
import { defineComponent, type PropType } from "vue";
import type { SponsorStatus } from "@/types/evcc";

export default defineComponent({
	name: "SponsorTokenExpires",
	mixins: [formatter],
	props: {
		status: Object as PropType<SponsorStatus>,
		fromYaml: Boolean,
	},
	computed: {
		expiresSoon() {
			return this.status?.expiresSoon;
		},
		expiresAt() {
			return this.status?.expiresAt;
		},
		name() {
			return this.status?.name;
		},
		inXDays() {
			return this.expiresAt
				? this.fmtTimeAgo(new Date(this.expiresAt).getTime() - new Date().getTime())
				: "";
		},
		isTrial() {
			return this.name === TRIAL;
		},
	},
});
</script>
