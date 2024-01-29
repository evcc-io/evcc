<template>
	<VueMonacoEditor
		v-if="!failedLoading"
		class="editor"
		language="yaml"
		:theme="theme"
		:height="height"
		:options="options"
		:value="modelValue"
		@update:value="$emit('update:modelValue', $event)"
	>
		loading editor ...
	</VueMonacoEditor>
	<textarea
		v-else
		class="form-control"
		rows="20"
		:value="modelValue"
		@input="$emit('update:modelValue', $event.target.value)"
	/>
</template>

<script>
import { VueMonacoEditor, loader } from "@guolao/vue-monaco-editor";

const $html = document.querySelector("html");

export default {
	name: "Editor",
	props: {
		modelValue: String,
		height: String,
	},
	components: { VueMonacoEditor },
	emits: ["update:modelValue"],
	data() {
		return {
			theme: "vs",
			options: {
				automaticLayout: true,
				formatOnType: true,
				formatOnPaste: true,
				minimap: { enabled: false },
				showFoldingControls: "always",
				scrollBeyondLastLine: false,
			},
			failedLoading: false,
		};
	},
	mounted() {
		this.updateTheme();
		$html.addEventListener("themechange", this.updateTheme);
	},
	beforeMount() {
		loader.config({
			paths: { vs: "https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs" },
		});
		loader.init().catch(() => (this.failedLoading = true));
	},
	unmounted() {
		$html.removeEventListener("themechange", this.updateTheme);
	},
	methods: {
		updateTheme() {
			this.theme = $html.classList.contains("dark") ? "vs-dark" : "vs";
		},
	},
};
</script>

<style scoped>
.editor :global(.monaco-editor) {
	--vscode-editor-background: var(--evcc-box) !important;
	--vscode-editorGutter-background: var(--evcc-box-border) !important;
}
</style>
