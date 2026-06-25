<template>
	<div ref="host" class="editor" data-testid="yaml-editor">
		<div v-if="loading" class="editor-loading">{{ $t("config.editor.loading") }}</div>
	</div>
</template>

<script>
export default {
	name: "YamlEditor",
	props: {
		modelValue: String,
		errorLine: Number,
		removeKey: String,
		disabled: Boolean,
	},
	emits: ["update:modelValue"],
	data() {
		return {
			loading: true,
		};
	},
	watch: {
		modelValue(value) {
			this.editor?.setDoc(value || "");
		},
		errorLine(line) {
			this.editor?.setErrorLine(line || null);
		},
		disabled(value) {
			this.editor?.setReadOnly(value);
		},
	},
	created() {
		// kept off the reactive data() so Vue doesn't proxy the EditorView
		this.editor = null;
		this.destroyed = false;
	},
	async mounted() {
		const { createEditor } = await import("./codemirror");
		// component may have been torn down while the chunk loaded
		if (this.destroyed || !this.$refs.host) return;

		this.editor = createEditor({
			parent: this.$refs.host,
			doc: this.modelValue || "",
			readOnly: this.disabled,
			onChange: (value) => this.$emit("update:modelValue", value),
			getRemoveKey: () => this.removeKey,
		});
		this.loading = false;

		if (this.errorLine) this.editor.setErrorLine(this.errorLine);
	},
	unmounted() {
		this.destroyed = true;
		this.editor?.destroy();
	},
};
</script>

<style scoped>
.editor {
	border: 1px solid var(--bs-border-color);
}
.editor-loading {
	padding: 0.5rem 0.75rem;
	color: var(--bs-secondary-color);
}

/* editor chrome */
.editor :global(.cm-editor) {
	height: 100%;
	font-size: 13px;
	background-color: var(--evcc-box);
	color: var(--evcc-default-text);
}
.editor :global(.cm-editor.cm-focused) {
	outline: none;
}
.editor :global(.cm-content) {
	font-family: var(--bs-font-monospace, monospace);
}
/* override CM's built-in light gutter colors (we don't load CM's dark chrome) */
.editor :global(.cm-gutters) {
	background-color: var(--evcc-box-border) !important;
	color: var(--bs-secondary-color) !important;
	border: none;
	padding-left: 6px;
}
.editor :global(.cm-activeLine),
.editor :global(.cm-activeLineGutter) {
	background-color: transparent;
}
.editor :global(.cm-cursor) {
	border-left-color: var(--evcc-default-text) !important;
}
.editor :global(.cm-selectionBackground),
.editor :global(.cm-focused .cm-selectionBackground) {
	background-color: rgba(128, 128, 128, 0.4) !important;
}
.editor :global(.cm-errorLine) {
	background-color: var(--bs-danger) !important;
}
/* keep error text readable over the red regardless of syntax colors */
.editor :global(.cm-errorLine),
.editor :global(.cm-errorLine span) {
	color: var(--bs-white) !important;
}

/* syntax tokens (classHighlighter), bootstrap palette reads well on light and dark */
.editor :global(.tok-comment),
.editor :global(.tok-meta) {
	color: var(--bs-gray-medium);
}
.editor :global(.tok-propertyName),
.editor :global(.tok-keyword) {
	color: var(--bs-orange);
}
.editor :global(.tok-string),
.editor :global(.tok-string2) {
	color: var(--bs-green);
}
.editor :global(.tok-bool),
.editor :global(.tok-atom),
.editor :global(.tok-number) {
	color: var(--bs-purple);
}
.editor :global(.tok-invalid) {
	color: var(--bs-red);
}
</style>
