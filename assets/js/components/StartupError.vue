<template>
	<div class="container px-4">
		<div class="d-flex justify-content-between align-items-center my-3">
			<h1 class="d-block mt-0">evcc konnte nicht starten</h1>
		</div>
		<div class="row mb-4">
			<h5 class="mb-3">Fehlermeldung</h5>
			<code class="fs-6 mb-3">
				<div v-for="(error, index) in errors" :key="index">{{ error }}</div>
			</code>
			<p>
				Lorem ipsum dolor, sit amet consectetur adipisicing elit. Culpa, eius nesciunt enim
				ullam, atque voluptas dolor blanditiis fugiat possimus at nam modi ducimus
				reiciendis dicta molestias, libero minus ex accusantium?
			</p>
		</div>
		<div class="row mb-4">
			<h5 class="mb-3">Aktuelle Konfiguration</h5>
			<code v-if="config">
				<div class="my-2">
					{{ file }}<span v-if="line">:{{ line }}</span>
				</div>
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
import "@h2d2/shopicons/es/regular/arrowup";
import collector from "../mixins/collector";

export default {
	name: "StartupError",
	mixins: [collector],
	props: {
		fatal: Array,
		config: String,
		file: String,
		line: { type: Number, default: 6 },
	},
	computed: {
		errors() {
			return this.fatal || [];
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
</style>
