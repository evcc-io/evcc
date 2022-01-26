<template>
	<div class="app overflow-hidden">
		<div class="position-absolute top-0 end-0 d-flex px-3 py-4 align-items-center">
			<Notifications :notifications="notifications" class="me-2" />
			<button
				type="button"
				data-bs-toggle="dropdown"
				data-bs-target="#navbarNavAltMarkup"
				aria-controls="navbarNavAltMarkup"
				aria-expanded="false"
				aria-label="Toggle navigation"
				class="btn btn-sm btn-outline-secondary"
			>
				<shopicon-regular-menu></shopicon-regular-menu>
			</button>
			<ul class="dropdown-menu dropdown-menu-end">
				<li>
					<a class="dropdown-item" href="https://docs.evcc.io/blog/" target="_blank">
						{{ $t("header.blog") }}
					</a>
				</li>
				<li>
					<a class="dropdown-item" href="https://docs.evcc.io/docs/Home/" target="_blank">
						{{ $t("header.docs") }}
					</a>
				</li>
				<li>
					<a class="dropdown-item" href="https://github.com/evcc-io/evcc" target="_blank">
						{{ $t("header.github") }}
					</a>
				</li>
			</ul>
		</div>
		<router-view class="flex-grow-1 d-flex flex-column justify-content-stretch"></router-view>
	</div>
</template>

<script>
import Notifications from "../components/Notifications";
import "@h2d2/shopicons/es/regular/menu";

import store from "../store";

export default {
	name: "App",
	components: { Notifications },
	props: {
		notifications: Array,
	},
	data: function () {
		return {
			store: this.$root.$data.store,
		};
	},
	computed: {
		title: function () {
			return this.store.state.siteTitle;
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
	background-color: var(--bs-gray-dark);
}
</style>
