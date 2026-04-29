<template>
	<Teleport to="body">
		<div class="more-backdrop" :class="{ open }" @click="$emit('close')"></div>
	</Teleport>
	<div class="more-menu dropdown-menu" :class="{ open }" @click.stop="$emit('close')">
		<button type="button" class="dropdown-item" @click="openHelpModal">
			{{ $t("header.needHelp") }}
		</button>
		<button
			type="button"
			class="dropdown-item d-flex align-items-center"
			@click="openAboutModal"
		>
			<span v-if="showVersionBadge" class="circle-badge me-1 bg-darker-green"></span>
			<span>evcc</span>
			<span class="ms-2 text-muted small">{{ versionLabel }}</span>
			<shopicon-regular-gift
				v-if="newVersionAvailable"
				size="s"
				class="ms-2 text-gray-light gift-icon"
			></shopicon-regular-gift>
		</button>
		<button v-if="isApp" type="button" class="dropdown-item" @click="openNativeSettings">
			{{ $t("header.nativeSettings") }}
		</button>
		<button v-if="showLogout" type="button" class="dropdown-item" @click="doLogout">
			{{ $t("header.logout") }}
		</button>
		<template v-if="authorizationRequired">
			<hr class="dropdown-divider" />
			<h6 class="dropdown-header">
				{{ $t("authProviders.authorizationRequired") }}
			</h6>
			<button
				v-for="provider in providers"
				:key="provider.id"
				type="button"
				class="dropdown-item"
				@click="handleAuthRequired"
			>
				<span
					class="d-inline-block p-1 rounded-circle border border-light bg-warning"
				></span>
				{{ provider.title }}
			</button>
		</template>
		<hr class="dropdown-divider" />
		<router-link class="dropdown-item" to="/log" active-class="active">
			{{ $t("log.title") }}
		</router-link>
		<button type="button" class="dropdown-item" @click="openSettingsModal">
			{{ $t("settings.title") }}
		</button>
		<router-link class="dropdown-item" to="/config" active-class="active">
			<span v-if="showConfigBadge" class="circle-badge me-1" :class="badgeClass"></span>
			{{ $t("config.main.title") }}
		</router-link>
		<router-link
			v-if="optimizeAvailable"
			class="dropdown-item"
			to="/optimize"
			active-class="active"
		>
			Optimize 🧪
		</router-link>
		<router-link v-if="experimental" class="dropdown-item" to="/history" active-class="active">
			History 🧪
		</router-link>
	</div>
</template>

<script lang="ts">
import Modal from "bootstrap/js/dist/modal";
import "@h2d2/shopicons/es/regular/gift";
import { logout, isLoggedIn } from "../Auth/auth";
import { isApp, sendToApp } from "@/utils/native";
import {
	getShortVersion,
	isNewVersionAvailable,
	isNewVersionUnacknowledged,
} from "@/utils/version";
import settings from "@/settings";
import { isUserConfigError } from "@/utils/fatal";
import { defineComponent, type PropType } from "vue";
import type { FatalError, Sponsor, EvOpt, AuthProviders } from "@/types/evcc";

export default defineComponent({
	name: "MoreMenu",
	props: {
		open: { type: Boolean, default: false },
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
	emits: ["close"],
	data() {
		return {
			isApp: isApp(),
			onClickOutside: undefined as ((e: MouseEvent) => void) | undefined,
		};
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
		badgeClass() {
			if (this.fatal.length > 0) {
				return "bg-danger";
			}
			return "bg-warning";
		},
		versionLabel() {
			return getShortVersion(this.installed || "", this.commit);
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
		optimizeAvailable() {
			return !!this.evopt && this.experimental;
		},
		showLogout() {
			return !this.authDisabled && isLoggedIn();
		},
	},
	mounted() {
		this.onClickOutside = (e: MouseEvent) => {
			if (this.open && !this.$el.contains(e.target as Node)) {
				this.$emit("close");
			}
		};
		document.addEventListener("click", this.onClickOutside, true);
	},
	unmounted() {
		if (this.onClickOutside) {
			document.removeEventListener("click", this.onClickOutside, true);
		}
	},
	methods: {
		handleAuthRequired() {
			this.$router.push({ path: "/config" });
		},
		openSettingsModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("globalSettingsModal") as HTMLElement
			);
			modal.show();
		},
		openHelpModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("helpModal") as HTMLElement
			);
			modal.show();
		},
		openAboutModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("aboutModal") as HTMLElement
			);
			modal.show();
		},
		openNativeSettings() {
			sendToApp({ type: "settings" });
		},
		async doLogout() {
			await logout();
			this.$router.push({ path: "/" });
		},
	},
});
</script>

<style scoped>
.more-backdrop {
	position: fixed;
	inset: 0;
	z-index: 1029;
	background-color: var(--evcc-backdrop);
	backdrop-filter: var(--evcc-backdrop-blur);
	opacity: 0;
	visibility: hidden;
	transition:
		opacity var(--evcc-transition-fast),
		visibility var(--evcc-transition-fast);
}

.more-backdrop.open {
	opacity: 1;
	visibility: visible;
}

.more-menu {
	display: block;
	position: absolute;
	right: 15%;
	bottom: calc(100% + 0.5rem);
	min-width: 70%;
	-webkit-user-select: none;
	user-select: none;
	opacity: 0;
	visibility: hidden;
	transform: translateY(0.5rem);
	transition:
		opacity var(--evcc-transition-fast),
		transform var(--evcc-transition-fast),
		visibility var(--evcc-transition-fast);
	background: color-mix(in srgb, var(--tab-bar-background) 80%, transparent);
}

.more-menu.open {
	opacity: 1;
	visibility: visible;
	transform: translateY(0);
}

:root.dark .more-menu {
	border: 1px solid var(--bs-border-color);
}

.dropdown-item.active,
.dropdown-item.router-link-active {
	background-color: transparent;
	color: var(--bs-primary);
	border-left: 2px solid var(--bs-primary);
}

.gift-icon {
	position: relative;
	top: -2px;
}
</style>
