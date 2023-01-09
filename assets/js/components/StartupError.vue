<template>
	<div class="container px-4">
		<div class="d-flex justify-content-between align-items-center my-3">
			<h1 class="d-block mt-0 d-flex">
				{{ $t("startupError.title") }}
				<shopicon-regular-car1 size="m" class="ms-2 icon"></shopicon-regular-car1>
			</h1>
		</div>
		<div class="row mb-4">
			<code class="fs-6 mb-3">
				<div v-for="(error, index) in errors" :key="index">{{ error }}</div>
			</code>
			<i18n-t tag="p" keypath="startupError.description">
				<a href="https://github.com/evcc-io/evcc/discussions">
					{{ $t("startupError.discussions") }}
				</a>
			</i18n-t>
			<p>
				<em>{{ $t("startupError.hint") }}</em>
			</p>
		</div>
		<div class="row mb-4">
			<h5 class="mb-3">{{ $t("startupError.configuration") }}</h5>
			<div class="d-md-flex justify-content-between">
				<p class="me-md-4">
					<span class="d-block">
						{{ $t("startupError.configFile") }}
						<code>{{ file }}</code>
					</span>
					<i18n-t v-if="line" tag="span" keypath="startupError.lineError">
						<a :href="`#line${line}`" @click.prevent="scrollTo">{{
							$t("startupError.lineErrorLink", [line])
						}}</a>
					</i18n-t>
					{{ $t("startupError.fixAndRestart") }}
				</p>
				<p>
					<button
						class="btn btn-primary text-nowrap"
						type="button"
						:disabled="offline"
						@click="shutdown"
					>
						{{ $t("startupError.restartButton") }}
					</button>
				</p>
			</div>

			<code v-if="config">
				<div class="my-2"></div>
				<div class="py-2 text-muted config">
					<div
						v-for="(configLine, lineNumber) in config.split('\n')"
						:id="`line${lineNumber + 1}`"
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
		offline: Boolean,
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
		scrollTo(e) {
			const id = e.currentTarget.getAttribute("href").substring(1);
			const el = document.getElementById(id);
			if (el) {
				el.scrollIntoView({ behavior: "smooth", block: "center" });
			}
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
