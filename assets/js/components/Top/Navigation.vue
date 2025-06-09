<template>
	<div>
		<button
			id="topNavigatonDropdown"
			type="button"
			data-bs-toggle="dropdown"
			aria-expanded="false"
			class="btn btn-sm btn-outline-secondary position-relative border-0 menu-button"
			data-testid="topnavigation-button"
		>
			<span
				v-if="showBadge"
				class="position-absolute top-0 start-100 translate-middle p-2 rounded-circle"
				:class="badgeClass"
			>
				<span class="visually-hidden">action required</span>
			</span>
			<shopicon-regular-menu></shopicon-regular-menu>
		</button>
		<ul
			class="dropdown-menu dropdown-menu-end"
			aria-labelledby="topNavigatonDropdown"
			data-testid="topnavigation-dropdown"
		>
			<li>
				<router-link class="dropdown-item" to="/sessions" active-class="active">
					{{ $t("header.sessions") }}
				</router-link>
			</li>
			<li><hr class="dropdown-divider" /></li>
			<li>
				<button
					type="button"
					class="dropdown-item"
					data-testid="topnavigation-settings"
					@click="openSettingsModal"
				>
					{{ $t("settings.title") }}
				</button>
			</li>
			<li v-if="batteryModalAvailable">
				<button
					type="button"
					class="dropdown-item"
					data-testid="topnavigation-battery"
					@click="openBatterySettingsModal"
				>
					{{ $t("batterySettings.modalTitle") }}
				</button>
			</li>
			<li v-if="forecastAvailable">
				<button
					type="button"
					class="dropdown-item"
					data-testid="topnavigation-forecast"
					@click="openForecastModal"
				>
					{{ $t("forecast.modalTitle") }}
				</button>
			</li>
			<li>
				<router-link class="dropdown-item" to="/config" active-class="active">
					<span
						v-if="showBadge"
						class="d-inline-block p-1 rounded-circle bg-warning rounded-circle"
						:class="badgeClass"
					></span>
					{{ $t("config.main.title") }}
				</router-link>
			</li>
			<li>
				<router-link class="dropdown-item" to="/log" active-class="active">
					{{ $t("log.title") }}
				</router-link>
			</li>
			<li><hr class="dropdown-divider" /></li>
			<template v-if="providerLogins.length > 0">
				<li><hr class="dropdown-divider" /></li>
				<li>
					<h6 class="dropdown-header">{{ $t("header.login") }}</h6>
				</li>
				<li v-for="l in providerLogins" :key="l.title">
					<button
						type="button"
						class="dropdown-item"
						@click="handleProviderAuthorization(l)"
					>
						<span
							v-if="!l.loggedIn"
							class="d-inline-block p-1 rounded-circle border border-light rounded-circle"
							:class="badgeClass"
						></span>
						{{ l.title }}
						{{ $t(l.loggedIn ? "main.provider.logout" : "main.provider.login") }}
					</button>
				</li>
			</template>
			<li>
				<button type="button" class="dropdown-item" @click="openHelpModal">
					<span>{{ $t("header.needHelp") }}</span>
				</button>
			</li>
			<li>
				<a class="dropdown-item d-flex" href="https://evcc.io/" target="_blank">
					<span>evcc.io</span>
					<shopicon-regular-newtab
						size="s"
						class="ms-2 external"
					></shopicon-regular-newtab>
				</a>
			</li>
			<li v-if="isApp">
				<button type="button" class="dropdown-item" @click="openNativeSettings">
					{{ $t("header.nativeSettings") }}
				</button>
			</li>
			<li v-if="showLogout">
				<button type="button" class="dropdown-item" @click="logout">
					{{ $t("header.logout") }}
				</button>
			</li>
		</ul>
	</div>
</template>

<script lang="ts">
import Modal from "bootstrap/js/dist/modal";
import Dropdown from "bootstrap/js/dist/dropdown";
import "@h2d2/shopicons/es/regular/gift";
import "@h2d2/shopicons/es/regular/moonstars";
import "@h2d2/shopicons/es/regular/menu";
import "@h2d2/shopicons/es/regular/newtab";
import collector from "@/mixins/collector";
import { logout, isLoggedIn, openLoginModal } from "../Auth/auth";
import baseAPI from "./baseapi";
import { isApp, sendToApp } from "@/utils/native";
import { isUserConfigError } from "@/utils/fatal";
import { defineComponent, type PropType } from "vue";
import type { FatalError, Sponsor, VehicleLogins } from "@/types/evcc";
import type { Provider as Provider } from "./types";

export default defineComponent({
	name: "TopNavigation",
	mixins: [collector],
	props: {
		vehicleLogins: {
			type: Object as PropType<VehicleLogins>,
			default: () => {
				return {};
			},
		},
		sponsor: {
			type: Object as PropType<Sponsor>,
			default: () => {
				return {};
			},
		},
		forecast: Object,
		battery: Array,
		fatal: Object as PropType<FatalError>,
	},
	data() {
		return {
			isApp: isApp(),
			dropdown: null as Dropdown | null,
		};
	},
	computed: {
		batteryConfigured() {
			return this.battery?.length;
		},
		logoutCount() {
			return this.providerLogins.filter((login) => !login.loggedIn).length;
		},
		providerLogins(): Provider[] {
			return Object.entries(this.vehicleLogins).map(([k, v]) => ({
				title: k,
				loggedIn: v.authenticated,
				loginPath: v.uri + "/login",
				logoutPath: v.uri + "/logout",
			}));
		},
		loginRequired() {
			return this.logoutCount > 0;
		},
		showBadge() {
			const userConfigError = isUserConfigError(this.fatal);
			return this.loginRequired || this.sponsor.expiresSoon || userConfigError;
		},
		badgeClass() {
			if (this.fatal?.error) {
				return "bg-danger";
			}
			return "bg-warning";
		},
		batteryModalAvailable() {
			return this.batteryConfigured;
		},
		forecastAvailable() {
			const { grid, solar, co2 } = this.forecast || {};
			return grid || solar || co2;
		},
		showLogout() {
			return isLoggedIn();
		},
	},
	mounted() {
		const $el = document.getElementById("topNavigatonDropdown");
		if (!$el) {
			return;
		}
		this.dropdown = new Dropdown(
			document.getElementById("topNavigatonDropdown") as HTMLElement
		);
	},
	unmounted() {
		this.dropdown?.dispose();
	},
	methods: {
		async handleProviderAuthorization(provider: Provider) {
			if (!provider.loggedIn) {
				baseAPI.post(provider.loginPath).then(function (response) {
					window.location.href = response.data.loginUri;
				});
			} else {
				baseAPI.post(provider.logoutPath);
			}
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
		openBatterySettingsModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("batterySettingsModal") as HTMLElement
			);
			modal.show();
		},
		openForecastModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("forecastModal") as HTMLElement
			);
			modal.show();
		},
		openNativeSettings() {
			sendToApp({ type: "settings" });
		},
		async login() {
			openLoginModal();
		},
		async logout() {
			await logout();
			this.$router.push({ path: "/" });
		},
	},
});
</script>
<style scoped>
.menu-button {
	margin-right: -0.7rem;
}
.external {
	width: 18px;
	height: 20px;
}
.dropdown-menu {
	/* above sticky, below modal https://getbootstrap.com/docs/5.3/layout/z-index/ */
	z-index: 1045 !important;
}
</style>
