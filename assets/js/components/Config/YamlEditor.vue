<template>
	<VueMonacoEditor
		v-if="active"
		ref="editor"
		class="editor"
		language="yaml"
		:theme="theme"
		:options="options"
		:value="modelValue"
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
import { cleanYaml } from "@/utils/cleanYaml.js";
// don't bundle monaco-editor but ensure that it get updated regularly
import { packages } from "../../../../package-lock.json";
const monacoVersion = packages["node_modules/monaco-editor"].version;

const $html = document.querySelector("html");
export default {
	name: "YamlEditor",
	components: { VueMonacoEditor },
	props: {
		modelValue: String,
		errorLine: Number,
		removeKey: String,
		disabled: Boolean,
	},
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
			pasteDisposable: null,
		};
	},
	computed: {
		options() {
			return { ...this.defaultOptions, readOnly: this.disabled };
		},
		lines() {
			return (this.modelValue || "").split("\n").length;
		},
	},
	watch: {
		errorLine() {
			// force rerender to update decorations
			this.active = false;
			this.$nextTick(() => (this.active = true));
		},
	},
	mounted() {
		this.updateTheme();
		$html.addEventListener("themechange", this.updateTheme);
	},
	beforeMount() {
		loader.config({
			paths: { vs: `https://cdn.jsdelivr.net/npm/monaco-editor@${monacoVersion}/min/vs` },
		});
		loader.init();
	},
	unmounted() {
		$html.removeEventListener("themechange", this.updateTheme);
		this.pasteDisposable?.dispose();
	},
	methods: {
		updateTheme() {
			this.theme = $html.classList.contains("dark") ? "vs-dark" : "vs";
		},
		ready(editor, monaco) {
			const disposable = editor.onDidPaste(async () => {
				if (!this.removeKey) return;
				await this.$nextTick();
				const model = editor.getModel();
				const cleaned = cleanYaml(model.getValue(), this.removeKey);
				model.setValue(cleaned);
			});

			this.pasteDisposable = disposable;

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
