<template>
	<div>
		<nav class="navbar navbar-expand-lg navbar-light bg-light">
			<a class="navbar-brand" href="https://github.com/andig/evcc"
				><fa-icon icon="leaf" class="text-primary mr-2"></fa-icon>evcc</a
			>
			<button
				class="navbar-toggler"
				type="button"
				data-toggle="collapse"
				data-target="#navbarNavAltMarkup"
				aria-controls="navbarNavAltMarkup"
				aria-expanded="false"
				aria-label="Toggle navigation"
			>
				<span class="navbar-toggler-icon"></span>
			</button>
			<div class="collapse navbar-collapse" id="navbarNavAltMarkup">
				<div class="navbar-nav">
					<router-link class="nav-item nav-link pb-1" to="/">Laden</router-link>
					<router-link class="nav-item nav-link pb-1" to="/config"
						>Konfiguration</router-link
					>
					<a
						class="nav-item nav-link pb-1"
						href="https://github.com/andig/evcc/discussions"
						target="_blank"
						>Community Support</a
					>
				</div>
			</div>
		</nav>

		<Version
			id="version-bar"
			:installed="installedVersion"
			:available="store.state.availableVersion"
			:releaseNotes="store.state.releaseNotes"
			:hasUpdater="store.state.hasUpdater"
			:uploadMessage="store.state.uploadMessage"
			:uploadProgress="store.state.uploadProgress"
		></Version>

		<router-view></router-view>
	</div>
</template>

<script>
import "../icons";
import Version from "../components/Version";
import store from "../store";

export default {
	name: "App",
	components: { Version },
	data: function () {
		return {
			compact: false,
			store: this.$root.$data.store,
			installedVersion: window.evcc.version,
		};
	},
	methods: {
		connect: function () {
			const loc = window.location;
			const protocol = loc.protocol == "https:" ? "wss:" : "ws:";
			const uri =
				protocol +
				"//" +
				loc.hostname +
				(loc.port ? ":" + loc.port : "") +
				loc.pathname +
				"ws";
			const ws = new WebSocket(uri),
				self = this;
			ws.onerror = function () {
				ws.close();
			};
			ws.onclose = function () {
				window.setTimeout(self.connect, 1000);
			};
			ws.onmessage = function (evt) {
				try {
					var msg = JSON.parse(evt.data);
					store.update(msg);
				} catch (e) {
					window.toasts.error(e, evt.data);
				}
			};
		},
	},
	created: function () {
		const urlParams = new URLSearchParams(window.location.search);
		this.compact = urlParams.get("compact");
		this.connect(); // websocket listener
	},
};
</script>
