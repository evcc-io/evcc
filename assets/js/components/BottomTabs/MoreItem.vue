<template>
	<Item
		:active="active"
		:label="$t('tabBar.more')"
		:badge="showRootBadge ? badgeClass : undefined"
		data-testid="tab-more"
		role="button"
		@click="toggleMenu"
	>
		<MoreIcon class="tab-icon d-block" />
		<template #menu>
			<MoreMenu
				:open="open"
				:auth-providers="authProviders"
				:sponsor="sponsor"
				:fatal="fatal"
				:experimental="experimental"
				:auth-disabled="authDisabled"
				:evopt="evopt"
				:installed="installed"
				:commit="commit"
				:available-version="availableVersion"
				@close="open = false"
			/>
		</template>
	</Item>
</template>

<script lang="ts">
import { defineComponent, type PropType } from "vue";
import Item from "./Item.vue";
import MoreIcon from "../MaterialIcon/More.vue";
import MoreMenu from "./MoreMenu.vue";
import { isUserConfigError } from "@/utils/fatal";
import { isNewVersionAvailable, isNewVersionUnacknowledged } from "@/utils/version";
import settings from "@/settings";
import type { FatalError, Sponsor, EvOpt, AuthProviders } from "@/types/evcc";

export default defineComponent({
	name: "MoreItem",
	components: { Item, MoreIcon, MoreMenu },
	props: {
		active: Boolean,
		authProviders: { type: Object as PropType<AuthProviders>, default: () => ({}) },
		sponsor: { type: Object as PropType<Sponsor>, default: () => ({}) },
		fatal: { type: Array as PropType<FatalError[]>, default: () => [] },
		experimental: Boolean,
		authDisabled: Boolean,
		evopt: { type: Object as PropType<EvOpt>, required: false },
		installed: String,
		commit: String,
		availableVersion: String,
	},
	data() {
		return { open: false };
	},
	computed: {
		providers() {
			return Object.entries(this.authProviders)
				.filter(([, provider]) => !provider.authenticated)
				.map(([title, { authenticated, id }]) => ({
					title,
					authenticated,
					id,
				}));
		},
		authorizationRequired() {
			return this.providers.length > 0;
		},
		sponsorExpires(): boolean {
			return !!this.sponsor?.status?.expiresSoon;
		},
		showConfigBadge() {
			return this.sponsorExpires || isUserConfigError(this.fatal);
		},
		newVersionAvailable() {
			return isNewVersionAvailable(this.installed, this.availableVersion);
		},
		showVersionBadge() {
			return isNewVersionUnacknowledged(
				this.installed,
				this.availableVersion,
				settings.lastAcknowledgedVersion
			);
		},
		showRootBadge() {
			return this.authorizationRequired || this.showConfigBadge || this.showVersionBadge;
		},
		badgeClass() {
			if (this.fatal.length > 0) {
				return "bg-danger";
			}
			if (this.authorizationRequired || this.showConfigBadge) {
				return "bg-warning";
			}
			return "bg-darker-green";
		},
	},
	methods: {
		toggleMenu() {
			this.open = !this.open;
		},
	},
});
</script>
