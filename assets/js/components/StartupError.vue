<template>
	<div class="container px-4">
		<div class="d-flex justify-content-between align-items-center my-3">
			<h1 class="d-block mt-0 d-flex">
				Fehler beim Starten
				<shopicon-regular-car1 size="m" class="ms-2 icon"></shopicon-regular-car1>
			</h1>
		</div>
		<div class="row mb-4">
			<code class="fs-6 mb-3">
				<div v-for="(error, index) in errors" :key="index">{{ error }}</div>
			</code>
			<p>
				Bitte überprüfe deine Konfigurationsdatei. Sollte dir die Fehlermeldung nicht
				weiterhelfen, suche in unseren
				<a href="https://github.com/evcc-io/evcc/discussions">GitHub Discussions</a> nach
				einer Lösung.
			</p>
			<p>
				<em>
					Hinweis: Ein weiterer Grund, warum du diese Meldung siehst, könnte ein
					fehlerhaftes Gerät (Wechselrichter, Zähler, ...) sein. Überprüfe deine
					Netzwerkverbindungen.
				</em>
			</p>
		</div>
		<div class="row mb-4">
			<h5 class="mb-3">Konfiguration</h5>
			<div class="d-md-flex justify-content-between">
				<p class="me-md-4">
					Folgende Konfigurationsdatei wurde verwendet:
					<code>
						{{ file }}<span v-if="line">:{{ line }}</span>
					</code>
					<br />
					Klicke hier um evcc neu zu starten nachdem du die Datei angepasst hast.
				</p>
				<p>
					<button
						class="btn btn-outline-primary text-nowrap"
						type="button"
						@click="shutdown"
					>
						Server neu starten
					</button>
				</p>
			</div>

			<code v-if="config">
				<div class="my-2"></div>
				<div class="py-2 text-muted config">
					<div
						v-for="(configLine, lineNumber) in config.split('\n')"
						:key="lineNumber"
						class="m-0 px-2"
						:class="{
							highlighted: line === lineNumber + 1,
						}"
					>
						{{ configLine }}&nbsp;
					</div>
				</div>
			</code>
		</div>
	</div>
</template>

<script>
import "@h2d2/shopicons/es/regular/car1";
import api from "../api";
import collector from "../mixins/collector";

export default {
	name: "StartupError",
	mixins: [collector],
	props: {
		fatal: Array,
		config: String,
		file: String,
		line: Number,
	},
	computed: {
		errors() {
			return this.fatal || [];
		},
	},
	methods: {
		shutdown() {
			api.post("shutdown");
		},
	},
};
</script>
<style scoped>
.highlighted {
	position: relative;
	color: var(--bs-code-color) !important;
}
.highlighted:before {
	content: "";
	position: absolute;
	left: 0;
	top: 0.2em;
	border-top: 0.5em solid transparent;
	border-bottom: 0.5em solid transparent;
	border-left: 0.5em solid var(--bs-code-color);
}
.config {
	max-width: 100%;
	overflow-x: scroll;
	white-space: pre;
}
.config {
	border: 1px solid var(--bs-gray-400);
}
.icon {
	transform-origin: 60% 40%;
	animation: swinging 3.5s ease-in-out infinite;
}

@keyframes swinging {
	0% {
		transform: translateY(8%) rotate(170deg);
	}
	50% {
		transform: translateY(8%) rotate(185deg);
	}
	100% {
		transform: translateY(8%) rotate(170deg);
	}
}
</style>
