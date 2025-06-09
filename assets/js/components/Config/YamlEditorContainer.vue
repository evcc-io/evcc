<template>
	<div class="editor-container" :style="{ height: computedHeight }">
		<YamlEditor
			v-if="!hidden"
			v-model="localValue"
			class="editor"
			:errorLine="errorLine"
			:removeKey="removeKey"
		/>
	</div>
</template>

<script>
import YamlEditor from "./YamlEditor.vue";

export default {
	name: "YamlEditorContainer",
	components: { YamlEditor },
	props: {
		modelValue: String,
		errorLine: { type: [Number, null], default: null },
		removeKey: String,
		hidden: Boolean,
	},
	emits: ["update:modelValue"],
	data() {
		return {
			localValue: this.modelValue,
		};
	},
	computed: {
		computedHeight() {
			return Math.max(150, (this.localValue || "").split("\n").length * 18) + 22 + "px";
		},
	},
	watch: {
		modelValue: {
			handler(newVal) {
				if (this.localValue !== newVal) {
					this.localValue = newVal;
				}
			},
			immediate: true,
		},
		localValue: {
			handler(newVal) {
				this.$emit("update:modelValue", newVal);
			},
		},
	},
};
</script>

<style scoped>
.editor-container {
	width: 100%;
	overflow: hidden;
	margin: 0 -1rem 0 -1.25rem;
}
/* reset margins on lg */
@media (min-width: 992px) {
	.editor-container {
		margin: 0;
	}
}
</style>
