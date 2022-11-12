<template>
	<div>
		<button
			type="button"
			data-bs-toggle="dropdown"
			data-bs-target="#navbarNavAltMarkup"
			aria-controls="navbarNavAltMarkup"
			aria-expanded="false"
			aria-label="Toggle navigation"
			class="btn btn-sm btn-outline-secondary position-relative border-0 menu-button"
		>
			<span
				v-if="logoutCount > 0"
				class="position-absolute top-0 start-100 translate-middle p-2 bg-danger border border-light rounded-circle"
			>
				<span class="visually-hidden">login available</span>
			</span>
			<shopicon-regular-menu></shopicon-regular-menu>
		</button>
		<ul class="dropdown-menu dropdown-menu-end">
			<li>
				<router-link class="dropdown-item" to="/sessions">
					{{ $t("header.sessions") }}
				</router-link>
			</li>

			<li>
				<button type="button" class="dropdown-item" @click.stop="toggleTheme">
					{{ $t(`header.theme.${theme}`) }}
				</button>
			</li>
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
				<a class="dropdown-item d-flex" href="https://docs.evcc.io/blog/" target="_blank">
					<span>{{ $t("header.blog") }}</span>
					<shopicon-regular-newtab
						size="s"
						class="ms-2 external"
					></shopicon-regular-newtab>
				</a>
			</li>
			<li>
				<a
					class="dropdown-item d-flex"
					href="https://docs.evcc.io/docs/Home/"
					target="_blank"
				>
					<span>{{ $t("header.docs") }}</span>
					<shopicon-regular-newtab
						size="s"
						class="ms-2 external"
					></shopicon-regular-newtab>
				</a>
			</li>
			<li>
				<a
					class="dropdown-item d-flex"
					href="https://github.com/evcc-io/evcc"
					target="_blank"
				>
					<span>{{ $t("header.github") }}</span>
					<shopicon-regular-newtab
						size="s"
						class="ms-2 external"
					></shopicon-regular-newtab>
				</a>
			</li>
			<li>
				<a class="dropdown-item d-flex" href="https://evcc.io/" target="_blank">
					<span>{{ $t("header.about") }}</span>
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
import "@h2d2/shopicons/es/regular/menu";
import "@h2d2/shopicons/es/regular/newtab";

import baseAPI from "../baseapi";
import { getThemePreference, setThemePreference, THEMES } from "../theme";

export default {
	name: "TopNavigation",
	props: {
		vehicleLogins: {
			type: Object,
			default: () => {
				return {};
			},
		},
	},
	data: function () {
		return { theme: getThemePreference() };
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
	},
	methods: {
		toggleTheme: function () {
			const currentIndex = THEMES.indexOf(this.theme);
			const nextIndex = currentIndex < THEMES.length - 1 ? currentIndex + 1 : 0;
			this.theme = THEMES[nextIndex];
			setThemePreference(this.theme);
		},
		handleProviderAuthorization: async function (provider) {
			if (!provider.loggedIn) {
				baseAPI.post(provider.loginPath).then(function (response) {
					window.location.href = response.data.loginUri;
				});
			} else {
				baseAPI.post(provider.logoutPath);
			}
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
