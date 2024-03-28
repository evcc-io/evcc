<template>
	<div class="app">
		<router-view :notifications="notifications" :offline="offline"></router-view>

		<GlobalSettingsModal v-bind="globalSettingsProps" />
		<BatterySettingsModal v-if="batteryModalAvailabe" v-bind="batterySettingsProps" />
		<HelpModal />
		<PasswordModal />
		<LoginModal />
	</div>
</template>

<script>
import store from "../store";
import GlobalSettingsModal from "../components/GlobalSettingsModal.vue";
import BatterySettingsModal from "../components/BatterySettingsModal.vue";
import PasswordModal from "../components/PasswordModal.vue";
import LoginModal from "../components/LoginModal.vue";
import HelpModal from "../components/HelpModal.vue";
import collector from "../mixins/collector";
import { updateAuthStatus } from "../auth";

// assume offline if not data received for 60 seconds
let lastDataReceived = new Date();
const maxDataAge = 60 * 1000;
setInterval(() => {
	if (new Date() - lastDataReceived > maxDataAge) {
		console.log("no data received, assume we are offline");
		window.app.setOffline();
	}
}, 1000);

export default {
	name: "App",
	components: {
		GlobalSettingsModal,
		HelpModal,
		BatterySettingsModal,
		PasswordModal,
		LoginModal,
	},
	mixins: [collector],
	props: {
		notifications: Array,
		offline: Boolean,
	},
	data: () => {
		return { reconnectTimeout: null, ws: null, authNotConfigured: false };
	},
	head() {
		const siteTitle = store.state.siteTitle;
		return { title: siteTitle ? `${siteTitle} | evcc` : "evcc" };
	},
	watch: {
		version: function (prev, now) {
			if (!!prev && !!now) {
				console.log("new version detected. reloading browser", { now, prev });
				this.reload();
			}
		},
		offline: function () {
			updateAuthStatus();
		},
	},
	computed: {
		version: function () {
			return store.state.version;
		},
		batteryModalAvailabe: function () {
			return store.state.battery?.length;
		},
		globalSettingsProps: function () {
			return this.collectProps(GlobalSettingsModal, store.state);
		},
		batterySettingsProps() {
			return this.collectProps(BatterySettingsModal, store.state);
		},
	},
	mounted: function () {
		this.connect();
		document.addEventListener("visibilitychange", this.pageVisibilityChanged, false);
		updateAuthStatus();
	},
	unmounted: function () {
		this.disconnect();
		window.clearTimeout(this.reconnectTimeout);
		document.removeEventListener("visibilitychange", this.pageVisibilityChanged, false);
	},
	methods: {
		pageVisibilityChanged: function () {
			if (document.hidden) {
				window.clearTimeout(this.reconnectTimeout);
				this.disconnect();
			} else {
				this.connect();
				updateAuthStatus();
			}
		},
		reconnect: function () {
			window.clearTimeout(this.reconnectTimeout);
			this.reconnectTimeout = window.setTimeout(() => {
				this.disconnect();
				this.connect();
			}, 2500);
		},
		disconnect: function () {
			console.log("websocket disconnecting");
			if (this.ws) {
				this.ws.onerror = null;
				this.ws.onopen = null;
				this.ws.onclose = null;
				this.ws.onmessage = null;
				this.ws.close();
				this.ws = null;
			}
		},
		connect: function () {
			console.log("websocket connect");
			const supportsWebSockets = "WebSocket" in window;
			if (!supportsWebSockets) {
				window.app.raise({
					message: "Web sockets not supported. Please upgrade your browser.",
				});
				return;
			}

			if (this.ws) {
				console.log("websocket already connected");
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

			this.ws = new WebSocket(uri);
			this.ws.onerror = () => {
				console.error({ message: "Websocket error. Trying to reconnect." });
				this.ws.close();
			};
			this.ws.onopen = () => {
				console.log("websocket connected");
				window.app.setOnline();
			};
			this.ws.onclose = () => {
				console.log("websocket disconnected");
				window.app.setOffline();
				this.reconnect();
			};
			this.ws.onmessage = (evt) => {
				try {
					var msg = JSON.parse(evt.data);
					store.update(msg);
					lastDataReceived = new Date();
				} catch (error) {
					window.app.raise({
						message: `Failed to parse web socket data: ${error.message} [${evt.data}]`,
					});
				}
			};
		},
		reload() {
			window.location.reload();
		},
	},
};
</script>
<style scoped>
.app {
	min-height: 100vh;
	min-height: 100dvh;
}
</style>
