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
				class="position-absolute top-0 start-100 translate-middle p-2 bg-danger border border-light rounded-circle"
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
				<router-link class="dropdown-item" to="/sessions">
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
					<span
						v-if="sponsorTokenExpires"
						class="d-inline-block p-1 rounded-circle bg-danger border border-light rounded-circle"
					></span>
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
			<li v-if="gridModalAvailable">
				<button
					type="button"
					class="dropdown-item"
					data-testid="topnavigation-grid"
					@click="openGridSettingsModal"
				>
					{{ $t("gridSettings.modalTitle") }}
				</button>
			</li>
			<li v-if="$hiddenFeatures()">
				<router-link class="dropdown-item" to="/config">
					Device Configuration 🧪
				</router-link>
			</li>
			<li><hr class="dropdown-divider" /></li>
			<template v-if="providerLogins.length > 0">
				<li><hr class="dropdown-divider" /></li>
				<li>
					<h6 class="dropdown-header">{{ $t("header.login") }}</h6>
				</li>
				<li v-for="login in providerLogins" :key="login.title">
					<button
						type="button"
						class="dropdown-item"
						@click="handleProviderAuthorization(login)"
					>
						<span
							v-if="!login.loggedIn"
							class="d-inline-block p-1 rounded-circle bg-danger border border-light rounded-circle"
						></span>
						{{ login.title }}
						{{ $t(login.loggedIn ? "main.provider.logout" : "main.provider.login") }}
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
		</ul>
	</div>
</template>

<script>
import Modal from "bootstrap/js/dist/modal";
import Dropdown from "bootstrap/js/dist/dropdown";
import "@h2d2/shopicons/es/regular/gift";
import "@h2d2/shopicons/es/regular/moonstars";
import "@h2d2/shopicons/es/regular/menu";
import "@h2d2/shopicons/es/regular/newtab";
import collector from "../mixins/collector";

import baseAPI from "../baseapi";

export default {
	name: "TopNavigation",
	mixins: [collector],
	props: {
		vehicleLogins: {
			type: Object,
			default: () => {
				return {};
			},
		},
		sponsor: String,
		sponsorTokenExpires: Number,
		batteryConfigured: Boolean,
	},
	computed: {
		logoutCount() {
			return this.providerLogins.filter((login) => !login.loggedIn).length;
		},
		providerLogins() {
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
			return this.loginRequired || this.sponsorTokenExpires;
		},
		batteryModalAvailable() {
			return this.batteryConfigured;
		},
	},
	mounted() {
		const $el = document.getElementById("topNavigatonDropdown");
		if (!$el) {
			return;
		}
		this.dropdown = new Dropdown(document.getElementById("topNavigatonDropdown"));
	},
	unmounted() {
		this.dropdown?.dispose();
	},
	methods: {
		handleProviderAuthorization: async function (provider) {
			if (!provider.loggedIn) {
				baseAPI.post(provider.loginPath).then(function (response) {
					window.location.href = response.data.loginUri;
				});
			} else {
				baseAPI.post(provider.logoutPath);
			}
		},
		openSettingsModal() {
			const modal = Modal.getOrCreateInstance(document.getElementById("globalSettingsModal"));
			modal.show();
		},
		openHelpModal() {
			const modal = Modal.getOrCreateInstance(document.getElementById("helpModal"));
			modal.show();
		},
		openBatterySettingsModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("batterySettingsModal")
			);
			modal.show();
		},
	},
};
</script>
<style scoped>
.menu-button {
	margin-right: -0.7rem;
}
.external {
	width: 18px;
	height: 20px;
}
</style>
