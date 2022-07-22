<template>
	<div class="app overflow-hidden">
		<metainfo>
			<template #title="{ content }">{{ content ? `${content} | evcc` : `evcc` }}</template>
		</metainfo>
		<router-view :notifications="notifications" :offline="offline"></router-view>
	</div>
</template>

<script>
import store from "../store";

export default {
	name: "App",
	props: {
		notifications: Array,
		offline: Boolean,
	},
	data: () => {
		return { reconnectTimeout: null };
	},
	created: function () {
		const urlParams = new URLSearchParams(window.location.search);
		this.compact = urlParams.get("compact");
		setTimeout(this.connect, 0);
	},
	methods: {
		reconnect: function () {
			window.clearTimeout(this.reconnectTimeout);
			this.reconnectTimeout = window.setTimeout(this.connect, 1000);
		},
		connect: function () {
			console.log("connecting websocket");
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
				console.error({ message: "Websocket error. Trying to reconnect." });
				ws.close();
			};
			ws.onopen = () => {
				console.log("websocket connected");
				window.app.setOnline();
			};
			ws.onclose = () => {
				console.log("websocket disconnected");
				window.app.setOffline();
				this.reconnect();
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
		return { title: store.state.siteTitle || "" };
	},
};
</script>
<style scoped>
.app {
	min-height: 100vh;
}
</style>
