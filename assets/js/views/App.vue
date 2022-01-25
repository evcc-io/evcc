<template>
	<div class="app d-flex flex-column justify-content-between overflow-hidden">
		<div class="flex-grow-1 d-flex flex-column justify-content-between">
			<nav class="navbar navbar-expand-lg navbar-dark bg-dark">
				<div class="container">
					<a class="navbar-brand" href="https://evcc.io/" target="_blank">
						<Logo class="logo"></Logo>
					</a>
					<div class="d-flex flex-grow-1 justify-content-end">
						<Notifications :notifications="notifications" />
						<button
							class="navbar-toggler"
							type="button"
							data-bs-toggle="collapse"
							data-bs-target="#navbarNavAltMarkup"
							aria-controls="navbarNavAltMarkup"
							aria-expanded="false"
							aria-label="Toggle navigation"
						>
							<span class="navbar-toggler-icon"></span>
						</button>
					</div>
					<div id="navbarNavAltMarkup" class="collapse navbar-collapse flex-grow-0">
						<ul class="navbar-nav">
							<li v-if="providerLogins.length > 0" class="nav-item dropdown">
								<a
									class="nav-link dropdown-toggle"
									data-bs-toggle="dropdown"
									href="#"
									role="button"
									aria-expanded="false"
									>{{ $t("header.login") }}
									<span v-if="logoutCount > 0" class="badge bg-secondary">{{
										logoutCount
									}}</span>
								</a>
								<ul class="dropdown-menu">
									<li
										v-for="login in providerLogins"
										:key="login.title"
										class="dropdown-item"
									>
										<button
											class="dropdown-item"
											type="button"
											@click="handleProviderAuthorization(login)"
										>
											{{ login.title }}
											{{
												$t(
													login.loggedIn
														? "main.provider.logout"
														: "main.provider.login"
												)
											}}
										</button>
									</li>
								</ul>
							</li>
							<li class="nav-item">
								<a
									class="nav-link"
									href="https://docs.evcc.io/blog/"
									target="_blank"
								>
									{{ $t("header.blog") }}
								</a>
							</li>
							<li class="nav-item">
								<a
									class="nav-link"
									href="https://docs.evcc.io/docs/Home/"
									target="_blank"
								>
									{{ $t("header.docs") }}
								</a>
							</li>
							<li class="nav-item">
								<a
									class="nav-link"
									href="https://github.com/evcc-io/evcc"
									target="_blank"
								>
									{{ $t("header.github") }}
								</a>
							</li>
						</ul>
					</div>
				</div>
			</nav>
			<router-view
				class="flex-grow-1 d-flex flex-column justify-content-stretch"
			></router-view>
		</div>
		<Footer :version="version" :sponsor="sponsor" :savings="savings"></Footer>
	</div>
</template>

<script>
import "../icons";
import Logo from "../components/Logo";
import Footer from "../components/Footer";
import Notifications from "../components/Notifications";

import store from "../store";

import authAPI from "../authapi";

export default {
	name: "App",
	components: { Logo, Footer, Notifications },
	props: {
		notifications: Array,
	},
	data: function () {
		return {
			compact: false,
			store: this.$root.$data.store,
			installedVersion: window.evcc.version,
			commit: window.evcc.commit,
		};
	},
	computed: {
		version: function () {
			return {
				installed: this.installedVersion,
				available: this.store.state.availableVersion,
				releaseNotes: this.store.state.releaseNotes,
				hasUpdater: this.store.state.hasUpdater,
				uploadMessage: this.store.state.uploadMessage,
				uploadProgress: this.store.state.uploadProgress,
			};
		},
		title: function () {
			return this.store.state.siteTitle;
		},
		sponsor: function () {
			return this.store.state.sponsor;
		},
		savings: function () {
			return {
				since: this.store.state.savingsSince,
				totalCharged: this.store.state.savingsTotalCharged,
				gridCharged: this.store.state.savingsGridCharged,
				selfConsumptionCharged: this.store.state.savingsSelfConsumptionCharged,
				amount: this.store.state.savingsAmount,
				effectivePrice: this.store.state.savingsEffectivePrice,
				selfConsumptionPercent: this.store.state.savingsSelfConsumptionPercent,
				gridPrice: this.store.state.tariffGrid,
				feedInPrice: this.store.state.tariffFeedIn,
				currency: this.store.state.currency,
			};
		},
		logoutCount() {
			return this.providerLogins.filter((login) => !login.loggedIn).length;
		},
		providerLogins() {
			return this.store.state.auth
				? Object.entries(this.store.state.auth.vehicles).map(([k, v]) => ({
						title: k,
						loggedIn: v.authenticated,
						loginPath: v.uri + "/login",
						logoutPath: v.uri + "/logout",
				  }))
				: [];
		},
	},
	created: function () {
		const urlParams = new URLSearchParams(window.location.search);
		this.compact = urlParams.get("compact");
		setTimeout(this.connect, 0);
	},
	methods: {
		connect: function () {
			const supportsWebSockets = "WebSocket" in window;
			if (!supportsWebSockets) {
				window.app.error({
					message: "Web sockets not supported. Please upgrade your browser.",
				});
				return;
			}

			const loc = window.location;
			const protocol = loc.protocol == "https:" ? "wss:" : "ws:";
			const uri =
				protocol +
				"//" +
				loc.hostname +
				(loc.port ? ":" + loc.port : "") +
				loc.pathname +
				"ws";
			const ws = new WebSocket(uri);
			ws.onerror = () => {
				ws.close();
			};
			ws.onclose = () => {
				window.setTimeout(this.connect, 1000);
			};
			ws.onmessage = (evt) => {
				try {
					var msg = JSON.parse(evt.data);
					store.update(msg);
				} catch (error) {
					window.app.error({
						message: `Failed to parse web socket data: ${error.message} [${evt.data}]`,
					});
				}
			};
		},
		reload() {
			window.location.reload();
		},
		handleProviderAuthorization: async function (provider) {
			if (!provider.loggedIn) {
				authAPI.post(provider.loginPath).then(function (response) {
					window.location.href = response.data.loginUri;
				});
			} else {
				authAPI.post(provider.logoutPath);
			}
		},
	},
	metaInfo() {
		return {
			title: this.title ? `evcc | ${this.title}` : "evcc",
		};
	},
};
</script>
<style scoped>
.logo {
	width: 85px;
}
.app {
	min-height: 100vh;
}
.title {
	position: relative;
	top: 0.1rem;
}
</style>
