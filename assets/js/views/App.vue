<template>
	<div class="app">
		<PinLockOverlay v-if="showUiLock" @unlocked="onUiLockUnlocked" />
		<router-view
			v-if="showRoutes"
			:notifications="notifications"
			:offline="offline"
		></router-view>

		<GlobalSettingsModal v-bind="globalSettingsProps" />
		<BatterySettingsModal v-if="batteryModalAvailabe" v-bind="batterySettingsProps" />
		<ForecastModal v-bind="forecastModalProps" />
		<HelpModal />
		<PasswordModal />
		<LoginModal v-bind="loginModalProps" />
		<OfflineIndicator v-bind="offlineIndicatorProps" />
	</div>
</template>

<script lang="ts">
import store from "../store";
import GlobalSettingsModal from "../components/GlobalSettings/GlobalSettingsModal.vue";
import BatterySettingsModal from "../components/Battery/BatterySettingsModal.vue";
import ForecastModal from "../components/Forecast/ForecastModal.vue";
import OfflineIndicator from "../components/Footer/OfflineIndicator.vue";
import PasswordModal from "../components/Auth/PasswordModal.vue";
import PinLockOverlay from "../components/Auth/PinLockOverlay.vue";
import LoginModal from "../components/Auth/LoginModal.vue";
import api from "../api";
import HelpModal from "../components/HelpModal.vue";
import collector from "../mixins/collector";
import { defineComponent } from "vue";

const WS_OPEN_TIMEOUT_MS = 5000;
const WS_RETRY_PARAM = "wsRetry";

// assume offline if not data received for 5 minutes
let lastDataReceived = new Date();
const maxDataAge = 60 * 1000 * 5;
setInterval(() => {
	if (new Date().getTime() - lastDataReceived.getTime() > maxDataAge) {
		console.log("no data received, assume we are offline");
		window.app.setOffline();
	}
}, 1000);

export default defineComponent({
	name: "App",
	components: {
		GlobalSettingsModal,
		HelpModal,
		BatterySettingsModal,
		ForecastModal,
		PasswordModal,
		PinLockOverlay,
		LoginModal,
		OfflineIndicator,
	},
	mixins: [collector],
	props: {
		notifications: Array,
		offline: Boolean,
	},
	data: () => {
		return {
			reconnectTimeout: null as number | null,
			openTimeout: null as number | null,
			ws: null as WebSocket | null,
			authNotConfigured: false,
			showUiLock: false,
			idleTimer: null as number | null,
			refreshTimer: null as number | null,
			idleActivityHandler: null as (() => void) | null,
			uilockTimeoutMs: 0,
			lastActivityTs: 0,
		};
	},
	head() {
		return { title: "...", titleTemplate: "%s | evcc" };
	},
	computed: {
		version() {
			return store.state.version;
		},
		batteryModalAvailabe() {
			return store.state.battery?.devices?.length;
		},
		showRoutes() {
			return this.state.startupCompleted;
		},
		state() {
			const { state, uiLoadpoints } = store;
			return { ...state, uiLoadpoints: uiLoadpoints.value };
		},
		globalSettingsProps() {
			return this.collectProps(GlobalSettingsModal, this.state);
		},
		batterySettingsProps() {
			return this.collectProps(BatterySettingsModal, this.state);
		},
		offlineIndicatorProps() {
			return this.collectProps(OfflineIndicator, this.state);
		},
		forecastModalProps() {
			return this.collectProps(ForecastModal, this.state);
		},
		loginModalProps() {
			return this.collectProps(LoginModal, this.state);
		},
	},
	watch: {
		version(now, prev) {
			if (!!prev && !!now) {
				console.log("new version detected. reloading browser", { now, prev });
				this.reload();
			}
		},
		offline(offline) {
			store.offline(offline);
			if (offline) {
				this.reconnect();
			}
		},
	},
	mounted() {
		void this.bootstrapUiLock();
		document.addEventListener("visibilitychange", this.pageVisibilityChanged, false);
		window.addEventListener("pageshow", this.pageShowHandler);
	},
	unmounted() {
		this.disconnect();
		this.clearReconnectTimeout();
		this.stopIdleMonitor();
		document.removeEventListener("visibilitychange", this.pageVisibilityChanged, false);
		window.removeEventListener("pageshow", this.pageShowHandler);
	},
	methods: {
		async bootstrapUiLock() {
			try {
				const res = await api.get("auth/uilock/status", {
					validateStatus: (s) => s >= 200 && s < 500,
				});
				const data = res.data as {
					appliesToClient?: boolean;
					unlocked?: boolean;
					timeout?: number;
				};
				if (data.appliesToClient && !data.unlocked) {
					store.update({ startupCompleted: true });
					this.showUiLock = true;
				} else if (data.appliesToClient && data.unlocked && data.timeout) {
					this.startIdleMonitor(data.timeout);
				}
			} catch (e) {
				console.warn("uilock status", e);
			}
			this.connect();
		},
		async onUiLockUnlocked() {
			this.showUiLock = false;
			try {
				const res = await api.get("auth/uilock/status", {
					validateStatus: (s) => s >= 200 && s < 500,
				});
				const timeout = (res.data as { timeout?: number }).timeout;
				if (timeout) this.startIdleMonitor(timeout);
			} catch {
				// fall back to store value if available
				const timeout = store.state.uilock?.timeout as number | undefined;
				if (timeout) this.startIdleMonitor(timeout);
			}
		},
		startIdleMonitor(timeoutSec: number) {
			this.stopIdleMonitor();
			if (timeoutSec <= 0) return;

			this.uilockTimeoutMs = timeoutSec * 1000;
			this.lastActivityTs = Date.now();

			this.idleActivityHandler = () => {
				const now = Date.now();
				if (now - this.lastActivityTs < 1000) return;
				this.lastActivityTs = now;
				this.resetIdleTimer();
			};
			const events = ["mousedown", "keydown", "touchstart", "scroll"];
			events.forEach((e) =>
				document.addEventListener(e, this.idleActivityHandler!, { passive: true }),
			);

			this.resetIdleTimer();

			const refreshMs = Math.min(60_000, Math.max(30_000, this.uilockTimeoutMs / 3));
			this.refreshTimer = window.setInterval(() => {
				if (!this.showUiLock && Date.now() - this.lastActivityTs < this.uilockTimeoutMs) {
					api.get("auth/uilock/status", { validateStatus: (s: number) => s < 500 });
				}
			}, refreshMs);
		},
		resetIdleTimer() {
			if (this.idleTimer) window.clearTimeout(this.idleTimer);
			this.idleTimer = window.setTimeout(() => {
				this.showUiLock = true;
				this.stopIdleMonitor();
			}, this.uilockTimeoutMs);
		},
		stopIdleMonitor() {
			if (this.idleTimer) {
				window.clearTimeout(this.idleTimer);
				this.idleTimer = null;
			}
			if (this.refreshTimer) {
				window.clearInterval(this.refreshTimer);
				this.refreshTimer = null;
			}
			if (this.idleActivityHandler) {
				const events = ["mousedown", "keydown", "touchstart", "scroll"];
				events.forEach((e) => document.removeEventListener(e, this.idleActivityHandler!));
				this.idleActivityHandler = null;
			}
		},
		clearReconnectTimeout() {
			if (this.reconnectTimeout) {
				window.clearTimeout(this.reconnectTimeout);
			}
		},
		// Safari 26 bug: with hash fragment URLs the HTTP upgrade
		// request is sometimes silently dropped when serving from cache.
		// Recover by navigating without hash, once (wsRetry guards against loops).
		startOpenTimeout() {
			const url = new URL(window.location.href);
			if (url.searchParams.has(WS_RETRY_PARAM)) return;
			this.openTimeout = window.setTimeout(() => {
				console.warn("websocket open timeout, forcing navigation");
				this.ws?.close();
				url.hash = "";
				url.searchParams.set(WS_RETRY_PARAM, "true");
				window.location.href = url.href;
			}, WS_OPEN_TIMEOUT_MS);
		},
		clearOpenTimeout(success = false) {
			if (this.openTimeout) {
				window.clearTimeout(this.openTimeout);
				this.openTimeout = null;
			}
			if (success) {
				const url = new URL(window.location.href);
				if (url.searchParams.has(WS_RETRY_PARAM)) {
					console.warn("websocket open timeout recovered, clearing retry param");
					url.searchParams.delete(WS_RETRY_PARAM);
					window.history.replaceState(window.history.state, "", url.href);
				}
			}
		},
		pageShowHandler(event: PageTransitionEvent) {
			if (event.persisted) {
				this.clearReconnectTimeout();
				this.disconnect();
				this.connect();
			}
		},
		pageVisibilityChanged() {
			// disconnect in any case to ensure fresh connection
			this.clearReconnectTimeout();
			this.disconnect();
			if (!document.hidden) {
				this.connect();
			}
		},
		reconnect() {
			this.clearReconnectTimeout();
			this.reconnectTimeout = window.setTimeout(() => {
				this.disconnect();
				this.connect();
			}, 2500);
		},
		disconnect() {
			this.clearOpenTimeout();
			if (this.ws) {
				this.ws.onerror = null;
				this.ws.onopen = null;
				this.ws.onclose = null;
				this.ws.onmessage = null;
				this.ws.close();
				this.ws = null;
			}
		},
		connect() {
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

			const loc = new URL("ws", window.location.href);
			loc.protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
			this.ws = new WebSocket(loc.href);

			this.startOpenTimeout();

			this.ws.onerror = () => {
				console.log({ message: "Websocket error. Trying to reconnect." });
				this.clearOpenTimeout();
				this.ws?.close();
			};
			this.ws.onopen = () => {
				this.clearOpenTimeout(true);
				console.log("websocket connected");
				window.app.setOnline();
			};
			this.ws.onclose = () => {
				this.clearOpenTimeout();
				window.app.setOffline();
				this.reconnect();
			};
			this.ws.onmessage = (evt) => {
				try {
					const msg = JSON.parse(evt.data);
					if (msg.startupCompleted) {
						store.reset();
					}
					store.update(msg);
					lastDataReceived = new Date();
				} catch (error) {
					const e = error as Error;
					window.app.raise({
						message: `Failed to parse web socket data: ${e.message} [${evt.data}]`,
					});
				}
			};
		},
		reload() {
			window.location.reload();
		},
	},
});
</script>
<style scoped>
.app {
	min-height: 100vh;
	min-height: 100dvh;
}
</style>
