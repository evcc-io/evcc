<template>
	<div class="app d-flex flex-column justify-content-between">
		<div>
			<nav class="navbar navbar-expand-lg navbar-dark bg-dark">
				<div class="container">
					<a class="navbar-brand" href="https://github.com/andig/evcc">
						<Logo class="logo"></Logo>
					</a>
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
						<div class="navbar-nav mr-auto"></div>
						<div class="navbar-nav">
							<a
								class="nav-item nav-link"
								href="https://github.com/andig/evcc/discussions"
								target="_blank"
							>
								Support
							</a>
						</div>
					</div>
				</div>
			</nav>

			<router-view></router-view>
		</div>
		<Footer :version="version"></Footer>
	</div>
</template>

<script>
import "../icons";
import Logo from "../components/Logo";
import Footer from "../components/Footer";
import store from "../store";

export default {
	name: "App",
	components: { Logo, Footer },
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
	},
	created: function () {
		const urlParams = new URLSearchParams(window.location.search);
		this.compact = urlParams.get("compact");
		this.connect(); // websocket listener
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
</style>
