<template>
	<VueMonacoEditor
		v-if="active"
		class="editor"
		language="yaml"
		:theme="theme"
		:options="options"
		:value="modelValue"
		ref="editor"
		@update:value="$emit('update:modelValue', $event)"
		@mount="ready"
	>
		<template #default> {{ $t("config.editor.loading") }} </template>
		<template #failure>
			<textarea
				class="form-control"
				:rows="lines"
				:value="modelValue"
				:disabled="disabled"
				@input="$emit('update:modelValue', $event.target.value)"
			/>
		</template>
	</VueMonacoEditor>
</template>

<script>
import { VueMonacoEditor, loader } from "@guolao/vue-monaco-editor";
const $html = document.querySelector("html");
export default {
	name: "YamlEditor",
	props: {
		modelValue: String,
		errorLine: Number,
		disabled: Boolean,
	},
	components: { VueMonacoEditor },
	emits: ["update:modelValue"],
	data() {
		return {
			theme: "vs",
			defaultOptions: {
				automaticLayout: true,
				formatOnType: true,
				formatOnPaste: true,
				minimap: { enabled: false },
				showFoldingControls: "always",
				scrollBeyondLastLine: false,
				wordWrap: "off",
				wrappingStrategy: "advanced",
				overviewRulerLanes: 0,
			},
			active: true,
		};
	},
	mounted() {
		this.updateTheme();
		$html.addEventListener("themechange", this.updateTheme);
	},
	watch: {
		errorLine() {
			// force rerender to update decorations
			this.active = false;
			this.$nextTick(() => (this.active = true));
		},
	},
	computed: {
		options() {
			return { ...this.defaultOptions, readOnly: this.disabled };
		},
		lines() {
			return (this.modelValue || "").split("\n").length;
		},
	},
	beforeMount() {
		loader.config({
			paths: { vs: "https://cdn.jsdelivr.net/npm/monaco-editor@0.45.0/min/vs" },
		});
		loader.init();
	},
	unmounted() {
		$html.removeEventListener("themechange", this.updateTheme);
	},
	methods: {
		updateTheme() {
			this.theme = $html.classList.contains("dark") ? "vs-dark" : "vs";
		},
		ready(editor, monaco) {
			let decorations = null;
			const highlight = () => {
				decorations?.clear();
				if (this.errorLine) {
					decorations = editor.createDecorationsCollection([
						{
							range: new monaco.Range(this.errorLine, 1, this.errorLine, 1),
							options: {
								isWholeLine: true,
								className: "error",
								lineNumberClassName: "error",
								marginClassName: "error",
							},
						},
					]);
				}
			};
			editor.onDidChangeModelContent(() => highlight());
			highlight();
		},
	},
};
</script>
<style scoped>
.editor :global(.monaco-editor) {
	--vscode-editor-background: var(--evcc-box) !important;
	--vscode-editorGutter-background: var(--evcc-box-border) !important;
}
.editor :global(.error) {
	background-color: var(--bs-danger-50) !important;
}
.editor {
	border: 1px solid var(--bs-border-color);
}
</style>
