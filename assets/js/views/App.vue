<template>
	<div class="app overflow-hidden">
		<router-view class="flex-grow-1 d-flex flex-column justify-content-stretch"></router-view>
	</div>
</template>

<script>
import store from "../store";

export default {
	name: "App",
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
.app {
	min-height: 100vh;
	background-color: var(--bs-gray-dark);
}
</style>
