<template>
	<div class="editor-container">
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
	overflow: hidden;
}
</style>
